package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("Admin actions", func() {
	BeforeEach(runAdminTestsInDevOnly)

	Specify("kubernetesobjects list", func() {
		result, err := adminRequest("GET", "kubernetesobjects", "", nil, "kind=configmap", "namespace=openshift-machine-api")
		Expect(err).NotTo(HaveOccurred())
		obj := &unstructured.Unstructured{}
		err = obj.UnmarshalJSON(result)
		Expect(err).NotTo(HaveOccurred())

		// Build list of valid names from the kubeclient
		configMaps, err := Clients.Kubernetes.CoreV1().ConfigMaps("openshift-machine-api").List(metav1.ListOptions{})
		var validNames []string
		for _, c := range configMaps.Items {
			validNames = append(validNames, c.Name)
		}

		// Compare names from kubernetesobjects API with valid names from kubeclient
		objs, err := obj.ToList()
		Expect(err).NotTo(HaveOccurred())
		for _, o := range objs.Items {
			Expect(o.GetName()).To(BeElementOf(validNames))
		}
	})

	Specify("kubernetesobjects get", func() {
		// Get objects via kubernetesobjects
		result, err := adminRequest("GET", "kubernetesobjects", "", nil, "kind=configmap", "namespace=openshift-machine-api", "name=cluster-autoscaler-operator-leader")
		Expect(err).NotTo(HaveOccurred())
		obj := &unstructured.Unstructured{}
		err = obj.UnmarshalJSON(result)
		Expect(err).NotTo(HaveOccurred())

		// Confirm via kubeclient
		cm, err := Clients.Kubernetes.CoreV1().ConfigMaps("openshift-machine-api").Get("cluster-autoscaler-operator-leader", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(obj.GetNamespace()).To(BeEquivalentTo(cm.Namespace))
		Expect(obj.GetName()).To(BeEquivalentTo(cm.Name))
		Expect(obj.GetAnnotations()).To(BeEquivalentTo(cm.Annotations))
	})

	Specify("kubernetesobjects create/delete", func() {
		// Create new object and confirm via kubeclient
		_, err := adminRequest("POST", "kubernetesobjects", `{
			"kind": "ConfigMap",
			"apiVersion": "v1",
			"metadata": {
				"name": "e2e-test-configmap",
				"namespace": "default"
			},
			"data": {
				"keys": "image.public.key=771 \nrsa.public.key=42"
			}
		}`, nil)
		Expect(err).NotTo(HaveOccurred())
		cm, err := Clients.Kubernetes.CoreV1().ConfigMaps("default").Get("e2e-test-configmap", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Namespace).To(BeEquivalentTo("default"))
		Expect(cm.Name).To(BeEquivalentTo("e2e-test-configmap"))

		// Delete object and confirm via kubeclient
		_, err = adminRequest("DELETE", "kubernetesobjects", "", nil, "kind=configmap", "namespace=default", "name=e2e-test-configmap")
		Expect(err).NotTo(HaveOccurred())
		cm, err = Clients.Kubernetes.CoreV1().ConfigMaps("default").Get("e2e-test-configmap", metav1.GetOptions{})
		Expect(errors.IsNotFound(err)).To(Equal(true))
	})

	Specify("kubernetesobjects update", func() {
		// Create new object
		_, err := adminRequest("POST", "kubernetesobjects", `{
			"kind": "ConfigMap",
			"apiVersion": "v1",
			"metadata": {
				"name": "e2e-test-configmap",
				"namespace": "default"
			},
			"data": {
				"keys": "image.public.key=771 \nrsa.public.key=42"
			}
		}`, nil)
		Expect(err).NotTo(HaveOccurred())

		// Update existing object. The data.keys object has changed.
		_, err = adminRequest("POST", "kubernetesobjects", `{
			"kind": "ConfigMap",
			"apiVersion": "v1",
			"metadata": {
				"name": "e2e-test-configmap",
				"namespace": "default"
			},
			"data": {
				"keys": "image.public.key=987 \nrsa.public.key=12"
			}
		}`, nil)
		Expect(err).NotTo(HaveOccurred())

		// Confirm via kubeclient
		cm, err := Clients.Kubernetes.CoreV1().ConfigMaps("default").Get("e2e-test-configmap", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cm.Namespace).To(BeEquivalentTo("default"))
		Expect(cm.Name).To(BeEquivalentTo("e2e-test-configmap"))
		Expect(cm.Data["keys"]).To(BeEquivalentTo("image.public.key=987 \nrsa.public.key=12"))
	})
})
