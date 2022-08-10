package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[Admin API] CertificateSigningRequest action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	const prefix = "e2e-test-csr"
	const namespace = "openshift"
	const csrCount = 4

	It("should be able to approve one or multiple CSRs", func() {
		By("creating mock CSRs via Kubernetes API")
		for i := 0; i < csrCount; i++ {
			csr := mockCSR(prefix+strconv.Itoa(i), namespace)
			_, err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Create(context.Background(), csr, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		defer func() {
			By("deleting the mock CSRs via Kubernetes API")
			for i := 0; i < csrCount; i++ {
				err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Delete(context.Background(), prefix+strconv.Itoa(i), metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			}
		}()

		testCSRApproveOK(prefix+"0", namespace)
		testCSRMassApproveOK(prefix, namespace, csrCount)
	})
})

func testCSRApproveOK(objName, namespace string) {
	By("approving the CSR via RP admin API")
	params := url.Values{
		"csrName":    []string{objName},
		"approveAll": []string{"false"},
	}
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/approvecsr", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that the CSR was approved via Kubernetes API")
	testcsr, err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get(context.Background(), objName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	approved := false
	for _, condition := range testcsr.Status.Conditions {
		if condition.Type == certificatesv1.CertificateApproved {
			Expect(condition.Status).To(Equal(corev1.ConditionTrue))
			Expect(condition.Reason).To(Equal("AROSupportApprove"))
			Expect(condition.Message).To(Equal("This CSR was approved by ARO support personnel."))
			approved = true
		}
	}
	Expect(approved).Should(BeTrue())
}

func testCSRMassApproveOK(namePrefix, namespace string, csrCount int) {
	By("approving all CSRs via RP admin API")
	params := url.Values{
		"approveAll": []string{"true"},
	}
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/approvecsr", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that all CSRs were approved via Kubernetes API")
	for i := 1; i < csrCount; i++ {
		testcsr, err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get(context.Background(), namePrefix+strconv.Itoa(i), metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		approved := false
		for _, condition := range testcsr.Status.Conditions {
			if condition.Type == certificatesv1.CertificateApproved {
				Expect(condition.Status).To(Equal(corev1.ConditionTrue))
				Expect(condition.Reason).To(Equal("AROSupportApprove"))
				Expect(condition.Message).To(Equal("This CSR was approved by ARO support personnel."))
				approved = true
			}
		}

		Expect(approved).Should(BeTrue())
	}
}

func mockCSR(objName, namespace string) *certificatesv1.CertificateSigningRequest {
	csr := &certificatesv1.CertificateSigningRequest{
		// Username, UID, Groups will be injected by API server.
		TypeMeta: metav1.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: namespace,
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request:    []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQ1ZqQ0NBVDRDQVFBd0VURVBNQTBHQTFVRUF3d0dZVzVuWld4aE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRgpBQU9DQVE4QU1JSUJDZ0tDQVFFQTByczhJTHRHdTYxakx2dHhWTTJSVlRWMDNHWlJTWWw0dWluVWo4RElaWjBOCnR2MUZtRVFSd3VoaUZsOFEzcWl0Qm0wMUFSMkNJVXBGd2ZzSjZ4MXF3ckJzVkhZbGlBNVhwRVpZM3ExcGswSDQKM3Z3aGJlK1o2MVNrVHF5SVBYUUwrTWM5T1Nsbm0xb0R2N0NtSkZNMUlMRVI3QTVGZnZKOEdFRjJ6dHBoaUlFMwpub1dtdHNZb3JuT2wzc2lHQ2ZGZzR4Zmd4eW8ybmlneFNVekl1bXNnVm9PM2ttT0x1RVF6cXpkakJ3TFJXbWlECklmMXBMWnoyalVnald4UkhCM1gyWnVVV1d1T09PZnpXM01LaE8ybHEvZi9DdS8wYk83c0x0MCt3U2ZMSU91TFcKcW90blZtRmxMMytqTy82WDNDKzBERHk5aUtwbXJjVDBnWGZLemE1dHJRSURBUUFCb0FBd0RRWUpLb1pJaHZjTgpBUUVMQlFBRGdnRUJBR05WdmVIOGR4ZzNvK21VeVRkbmFjVmQ1N24zSkExdnZEU1JWREkyQTZ1eXN3ZFp1L1BVCkkwZXpZWFV0RVNnSk1IRmQycVVNMjNuNVJsSXJ3R0xuUXFISUh5VStWWHhsdnZsRnpNOVpEWllSTmU3QlJvYXgKQVlEdUI5STZXT3FYbkFvczFqRmxNUG5NbFpqdU5kSGxpT1BjTU1oNndLaTZzZFhpVStHYTJ2RUVLY01jSVUyRgpvU2djUWdMYTk0aEpacGk3ZnNMdm1OQUxoT045UHdNMGM1dVJVejV4T0dGMUtCbWRSeEgvbUNOS2JKYjFRQm1HCkkwYitEUEdaTktXTU0xMzhIQXdoV0tkNjVoVHdYOWl4V3ZHMkh4TG1WQzg0L1BHT0tWQW9FNkpsYWFHdTlQVmkKdjlOSjVaZlZrcXdCd0hKbzZXdk9xVlA3SVFjZmg3d0drWm89Ci0tLS0tRU5EIENFUlRJRklDQVRFIFJFUVVFU1QtLS0tLQo"),
			Usages:     []certificatesv1.KeyUsage{certificatesv1.UsageClientAuth},
			SignerName: "kubernetes.io/kube-apiserver-client",
		},
	}

	return csr
}
