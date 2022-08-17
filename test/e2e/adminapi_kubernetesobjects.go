package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
)

var _ = Describe("[Admin API] Kubernetes objects action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	const objName = "e2e-test-object"

	When("in a standard openshift namespace", func() {
		const namespace = "openshift"
		const containerName = "e2e-test-container-name"

		It("should be able to create, get, list, update and delete objects, but force delete is only allowed for pods", func() {
			defer func() {
				// When ran successfully this test should delete the object,
				// but we need to remove the object in case of failure
				// to allow us to run this test against the same cluster multiple times.
				By("deleting the config map via Kubernetes API")
				err := clients.Kubernetes.CoreV1().ConfigMaps(namespace).Delete(context.Background(), objName, metav1.DeleteOptions{})
				// On successfully we expect NotFound error
				if !kerrors.IsNotFound(err) {
					Expect(err).NotTo(HaveOccurred())
				}
				By("deleting the pod via Kubernetes API")
				err = clients.Kubernetes.CoreV1().Pods(namespace).Delete(context.Background(), objName, metav1.DeleteOptions{})
				// On successfully we expect NotFound error
				if !kerrors.IsNotFound(err) {
					Expect(err).NotTo(HaveOccurred())
				}
			}()

			testConfigMapCreateOK(objName, namespace)
			testConfigMapGetOK(objName, namespace)
			testConfigMapListOK(objName, namespace)
			testConfigMapUpdateOK(objName, namespace)
			testConfigMapForceDeleteForbidden(objName, namespace)
			testConfigMapDeleteOK(objName, namespace)
			testPodCreateOK(containerName, objName, namespace)
			testPodForceDeleteOK(objName, namespace)
		})

		testSecretOperationsForbidden(objName, namespace)
	})

	When("in a customer namespace", func() {
		const namespace = "e2e-test-namespace"

		It("should be able to get and list existing objects, but not update and delete or create new objects", func() {
			By("creating a test customer namespace via Kubernetes API")
			_, err := clients.Kubernetes.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			defer func() {
				By("deleting the test customer namespace via Kubernetes API")
				err := clients.Kubernetes.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// To avoid flakes, we need it to be completely deleted before we can use it again
				// in a separate run or in a separate It block
				By("waiting for the test customer namespace to be deleted")
				err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
					_, err := clients.Kubernetes.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
					if kerrors.IsNotFound(err) {
						return true, nil
					}
					if err != nil {
						log.Warn(err)
					}
					return false, nil // swallow error
				})
				Expect(err).NotTo(HaveOccurred())
			}()

			testConfigMapCreateOrUpdateForbidden("creating", objName, namespace)

			By("creating an object via Kubernetes API")
			_, err = clients.Kubernetes.CoreV1().ConfigMaps(namespace).Create(context.Background(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: objName},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			testConfigMapGetOK(objName, namespace)
			testConfigMapListOK(objName, namespace)
			testConfigMapCreateOrUpdateForbidden("updating", objName, namespace)
			testConfigMapDeleteForbidden(objName, namespace)
		})

		testSecretOperationsForbidden(objName, namespace)
	})
})

func testSecretOperationsForbidden(objName, namespace string) {
	It("should not be able to create a secret", func() {
		By("creating a new secret via RP admin API")
		obj := mockSecret(objName, namespace)
		var cloudErr api.CloudError
		resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", nil, obj, &cloudErr)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
	})

	It("should not be able to get a secret", func() {
		By("requesting a secret via RP admin API")
		params := url.Values{
			"kind":      []string{"secret"},
			"namespace": []string{namespace},
			"name":      []string{objName},
		}
		var cloudErr api.CloudError
		resp, err := adminRequest(context.Background(), http.MethodGet, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &cloudErr)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

		By("checking response for an error")
		Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
	})

	It("should not be able to get a list of secrets", func() {
		By("requesting a list of Secret objects via RP admin API")
		params := url.Values{
			"kind":      []string{"secret"},
			"namespace": []string{namespace},
		}
		var cloudErr api.CloudError
		resp, err := adminRequest(context.Background(), http.MethodGet, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &cloudErr)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

		By("checking response for an error")
		Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
	})

	It("should not be able to delete a secret", func() {
		By("deleting the secret via RP admin API")
		params := url.Values{
			"kind":      []string{"secret"},
			"namespace": []string{namespace},
			"name":      []string{objName},
		}
		var cloudErr api.CloudError
		resp, err := adminRequest(context.Background(), http.MethodDelete, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &cloudErr)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
	})
}

func testConfigMapCreateOK(objName, namespace string) {
	By("creating a new object via RP admin API")
	obj := mockConfigMap(objName, namespace)
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", nil, obj, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that the object was created via Kubernetes API")
	cm, err := clients.Kubernetes.CoreV1().ConfigMaps(namespace).Get(context.Background(), objName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(obj.Namespace).To(Equal(cm.Namespace))
	Expect(obj.Name).To(Equal(cm.Name))
	Expect(obj.Data).To(Equal(cm.Data))
}

func testConfigMapGetOK(objName, namespace string) {
	By("getting an object via RP admin API")
	params := url.Values{
		"kind":      []string{"configmap"},
		"namespace": []string{namespace},
		"name":      []string{objName},
	}
	var obj corev1.ConfigMap
	resp, err := adminRequest(context.Background(), http.MethodGet, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &obj)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("comparing it to the actual object retrived via Kubernetes API")
	cm, err := clients.Kubernetes.CoreV1().ConfigMaps(namespace).Get(context.Background(), objName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(obj.Namespace).To(Equal(cm.Namespace))
	Expect(obj.Name).To(Equal(cm.Name))
	Expect(obj.Data).To(Equal(cm.Data))
}

func testConfigMapListOK(objName, namespace string) {
	By("requesting a list of objects via RP admin API")
	params := url.Values{
		"kind":      []string{"configmap"},
		"namespace": []string{namespace},
	}
	var obj corev1.ConfigMapList
	resp, err := adminRequest(context.Background(), http.MethodGet, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &obj)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("comparing names from the list action response with names retrived via Kubernetes API")
	var names []string
	for _, o := range obj.Items {
		names = append(names, o.Name)
	}
	Expect(names).To(ContainElement(objName))
}

func testConfigMapUpdateOK(objName, namespace string) {
	By("updating the object via RP admin API")
	obj := mockConfigMap(objName, namespace)
	obj.Data["key"] = "new_value"

	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", nil, obj, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that the object changed via Kubernetes API")
	cm, err := clients.Kubernetes.CoreV1().ConfigMaps(namespace).Get(context.Background(), objName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(cm.Namespace).To(Equal(namespace))
	Expect(cm.Name).To(Equal(objName))
	Expect(cm.Data).To(Equal(map[string]string{"key": "new_value"}))
}

func testConfigMapForceDeleteForbidden(objName, namespace string) {
	By("force deleting the object via RP admin API")
	params := url.Values{
		"kind":      []string{"configmap"},
		"namespace": []string{namespace},
		"name":      []string{objName},
		"force":     []string{"true"},
	}
	var cloudErr api.CloudError
	resp, err := adminRequest(context.Background(), http.MethodDelete, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &cloudErr)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
}

func testConfigMapDeleteOK(objName, namespace string) {
	By("deleting the object via RP admin API")
	params := url.Values{
		"kind":      []string{"configmap"},
		"namespace": []string{namespace},
		"name":      []string{objName},
		"force":     []string{"false"},
	}
	resp, err := adminRequest(context.Background(), http.MethodDelete, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	// To avoid flakes, we need it to be completely deleted before we can use it again
	// in a separate run or in a separate It block
	By("waiting for the configmap to be deleted")
	err = wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		_, err = clients.Kubernetes.CoreV1().ConfigMaps(namespace).Get(context.Background(), objName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			log.Warn(err)
		}
		return false, nil // swallow error
	})
	Expect(err).NotTo(HaveOccurred())
}

func testConfigMapCreateOrUpdateForbidden(operation, objName, namespace string) {
	By(operation + " a new object via RP admin API")
	obj := mockConfigMap(objName, namespace)
	var cloudErr api.CloudError
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", nil, obj, &cloudErr)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
}

func testConfigMapDeleteForbidden(objName, namespace string) {
	By("deleting the object via RP admin API")
	params := url.Values{
		"kind":      []string{"configmap"},
		"namespace": []string{namespace},
		"name":      []string{objName},
		"force":     []string{"false"},
	}
	var cloudErr api.CloudError
	resp, err := adminRequest(context.Background(), http.MethodDelete, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, &cloudErr)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	Expect(cloudErr.Code).To(Equal(api.CloudErrorCodeForbidden))
}

func testPodCreateOK(containerName, objName, namespace string) {
	By("creating a new pod via RP admin API")
	obj := mockPod(containerName, objName, namespace)
	resp, err := adminRequest(context.Background(), http.MethodPost, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", nil, obj, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("checking that the pod was created via Kubernetes API")
	pod, err := clients.Kubernetes.CoreV1().Pods(namespace).Get(context.Background(), objName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(obj.Namespace).To(Equal(pod.Namespace))
	Expect(obj.Name).To(Equal(pod.Name))
	Expect(obj.Spec.Containers[0].Name).To(Equal(pod.Spec.Containers[0].Name))
}

func testPodForceDeleteOK(objName, namespace string) {
	By("deleting the object via RP admin API")
	params := url.Values{
		"kind":      []string{"pod"},
		"namespace": []string{namespace},
		"name":      []string{objName},
		"force":     []string{"true"},
	}
	resp, err := adminRequest(context.Background(), http.MethodDelete, "/admin"+resourceIDFromEnv()+"/kubernetesobjects", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	// To avoid flakes, we need it to be completely deleted before we can use it again
	// in a separate run or in a separate It block
	By("waiting for the pod to be deleted")
	err = wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
		_, err = clients.Kubernetes.CoreV1().Pods(namespace).Get(context.Background(), objName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			log.Warn(err)
		}
		return false, nil // swallow error
	})
	Expect(err).NotTo(HaveOccurred())
}

func mockSecret(name, namespace string) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func mockConfigMap(name, namespace string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"key": "value",
		},
	}
}
