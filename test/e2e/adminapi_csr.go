package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
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

	BeforeEach(func(ctx context.Context) {
		const csrdataStr = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQ3BEQ0NBWXdDQVFBd1h6RUxNQWtHQTFVRUJoTUNWVk14Q3pBSkJnTlZCQWdNQWtOUE1ROHdEUVlEVlFRSApEQVpFWlc1MlpYSXhFakFRQmdOVkJBb01DVTFwWTNKdmMyOW1kREVNTUFvR0ExVUVDd3dEUVZKUE1SQXdEZ1lEClZRUUREQWRsTW1VdVlYSnZNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQWxqWUUKcnFkU0hvV1p2MVdHSnN1RFZsaGExVU1BVnJiUk0xWjJHaWZJMzNETlBUWGFmQnN1QVI2ZGVCQVgyWmpybUozNQpuekNBZ0k5d1ltdlYwN3JtTEFYQlloRnJiTWtNN1pSU1ZFT01hL2ZXdlN5ZjJVQWxSdm5Jd0JmRkgwS1pRSGg5Cm5aV3RIZHQxSzRuZ3ZnM1NuQ3JEU0NBRUhsS2hoN3Jua1pyRkdrMldabFFoVklWUXFReFFzdmx3VStvWlhnNjQKdmpleDRuc3BZaXFXMERzakl6RzFsSEszWHczN3RGeWhNNzJ4SjByblBYVTRGWkJsWXUzWkVqOFVhSFBoTlcrdgpqZmg2c0hCbWFkcHpEMWRuNDJ4eXgrUGhOaCtKWTVVT3ZWWnR2MWx5UU44eEswL0VjK0Mvcm1mOWZPYmdFSkNVCm00Z3pFSXhhVGhCVURsN1JHd0lEQVFBQm9BQXdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBQnYvVHdUR0JvL20KcVJVK0djZ3Bsa3I1aDlKQVdSZjNNazV3Z1o0ZmlSZm85UEVaYUxJWkZYQ0V0elNHV3JZenFjbFpZQ3JuRmUySQpzdHdNUU8yb1pQUzNvcUVIcWs5Uk0rbzRUVmtkSldjY3hKV3RMY3JoTWRwVjVMc3VMam1qRS9jeDcrbEtUZkh1Cno0eDllYzJTajhnZmV3SFowZTkzZjFTT3ZhVGFMaTQrT3JkM3FTT0NyNE5ZSGhvVDJiM0pBUFpMSmkvVEFpb1gKOUxJNFJpVXNSSWlMUm45VDZidzczM0FLMkpNMXREWU9Tc0hXdmJrZ3FDOFlHMmpYUW9LNUpZOWdTN0V5TkF6NwpjT1plbkkwK2dVeE1leUlNN2I0S05YWFQ3NmxVdHZ5M2N3LzhwVmxQU01pTDFVZ2RpMXFZMDl0MW9FMmU4YnljCm5GdWhZOW5ERU53PQotLS0tLUVORCBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0K"

		csrDataEncoded := []byte(csrdataStr)
		csrDataDecoded := make([]byte, base64.StdEncoding.DecodedLen(len(csrDataEncoded)))
		csrDataLength, err := base64.StdEncoding.Decode(csrDataDecoded, csrDataEncoded)
		Expect(err).NotTo(HaveOccurred())
		csrData := csrDataDecoded[:csrDataLength]

		By("creating mock CSRs via Kubernetes API")
		for i := 0; i < csrCount; i++ {
			csr := mockCSR(prefix+strconv.Itoa(i), namespace, csrData)
			_, err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Create(ctx, csr, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterEach(func(ctx context.Context) {
		By("deleting the mock CSRs via Kubernetes API")
		for i := 0; i < csrCount; i++ {
			err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Delete(ctx, prefix+strconv.Itoa(i), metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("must be able to approve one CSRs", func(ctx context.Context) {
		testCSRApproveOK(ctx, prefix+"0", namespace)
	})

	It("must be able to approve multiple CSRs", func(ctx context.Context) {
		testCSRMassApproveOK(ctx, prefix, namespace, csrCount)
	})

})

func testCSRApproveOK(ctx context.Context, objName, namespace string) {
	By("approving the CSR via RP admin API")
	params := url.Values{
		"csrName": []string{objName},
	}
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/approvecsr", params, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that the CSR was approved via Kubernetes API")
	testcsr, err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get(ctx, objName, metav1.GetOptions{})
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

func testCSRMassApproveOK(ctx context.Context, namePrefix, namespace string, csrCount int) {
	By("approving all CSRs via RP admin API")
	resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/approvecsr", nil, true, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that all CSRs were approved via Kubernetes API")
	for i := 1; i < csrCount; i++ {
		testcsr, err := clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get(ctx, namePrefix+strconv.Itoa(i), metav1.GetOptions{})
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

func mockCSR(objName, namespace string, csrData []byte) *certificatesv1.CertificateSigningRequest {
	csr := &certificatesv1.CertificateSigningRequest{
		// Username, UID, Groups will be injected by API server.
		TypeMeta: metav1.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: namespace,
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request:    csrData,
			Usages:     []certificatesv1.KeyUsage{certificatesv1.UsageClientAuth},
			SignerName: "kubernetes.io/kube-apiserver-client",
		},
	}

	return csr
}
