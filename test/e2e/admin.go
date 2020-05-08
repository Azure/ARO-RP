package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func adminRequest(method, action string, body string, options ...string) ([]byte, error) {
	resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), os.Getenv("CLUSTER"))
	adminURL, err := url.Parse("https://localhost:8443/admin" + resourceID + "/" + action)
	if err != nil {
		return nil, err
	}
	q := adminURL.Query()
	for _, opt := range options {
		optSplit := strings.Split(opt, "=")
		q.Set(optSplit[0], optSplit[1])
	}
	adminURL.RawQuery = q.Encode()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	fmt.Println(adminURL.String())
	req, err := http.NewRequest(method, adminURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

var _ = Describe("Admin actions", func() {
	Specify("KubernetesObjects get/set/delete", func() {
		// 1. get with both kube client and KubernetesObjects
		result, err := adminRequest("GET", "kubernetesobjects", "", "kind=configmap", "namespace=openshift-machine-api", "name=cluster-autoscaler-operator-leader")
		Expect(err).NotTo(HaveOccurred())
		obj := &unstructured.Unstructured{}
		err = obj.UnmarshalJSON(result)
		Expect(err).NotTo(HaveOccurred())

		cm, err := Clients.Kubernetes.CoreV1().ConfigMaps("openshift-machine-api").Get("cluster-autoscaler-operator-leader", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(obj.GetNamespace()).To(BeEquivalentTo(cm.Namespace))
		Expect(obj.GetName()).To(BeEquivalentTo(cm.Name))
		Expect(obj.GetAnnotations()).To(BeEquivalentTo(cm.Annotations))

		// 2. create an object and confirm
		_, err = adminRequest("POST", "kubernetesobjects", `{
			"kind": "ConfigMap",
			"apiVersion": "v1",
			"metadata": {
				"name": "e2e-test-configmap",
				"namespace": "default"
			},
			"data": {
				"keys": "image.public.key=771 \nrsa.public.key=42"
			}
		}`)
		Expect(err).NotTo(HaveOccurred())
		cm, err = Clients.Kubernetes.CoreV1().ConfigMaps("default").Get("e2e-test-configmap", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Namespace).To(BeEquivalentTo("default"))
		Expect(cm.Name).To(BeEquivalentTo("e2e-test-configmap"))

		// 3. delete and confirm
		_, err = adminRequest("DELETE", "kubernetesobjects", "", "kind=configmap", "namespace=default", "name=e2e-test-configmap")
		Expect(err).NotTo(HaveOccurred())
		cm, err = Clients.Kubernetes.CoreV1().ConfigMaps("default").Get("e2e-test-configmap", metav1.GetOptions{})
		Expect(err).To(HaveOccurred())
	})
})
