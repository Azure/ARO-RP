package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	proj "github.com/Azure/ARO-RP/test/util/project"
)

const (
	testNamespace = "test-e2e"
	testPodName   = "busybox"
)

var _ = Describe("Cluster smoke test", func() {
	var project proj.Project

	var _ = BeforeEach(func() {
		project = proj.NewProject(clients.Kubernetes, clients.Project, testNamespace)
		err := project.Create()
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

		Eventually(func() error {
			return project.Verify()

		}).Should(BeNil())
	})

	var _ = AfterEach(func() {
		err := project.Delete()
		Expect(err).NotTo(HaveOccurred(), "Failed to delete test namespace")

		Eventually(func() error {
			return project.VerifyProjectIsDeleted()

		}, 5*time.Minute, 1*time.Second).Should(BeNil())
	})

	Specify("Can run a pod which is using Azure File storage", func() {
		ctx := context.Background()
		err := createPVC(ctx, clients.Kubernetes)
		Expect(err).NotTo(HaveOccurred())

		err = createPodWithPVC(ctx, clients.Kubernetes)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			return verifyPodSucceeded(clients.Kubernetes)

		}).Should(BeNil())
	})

	Specify("Can create load balancer services", func() {
		ctx := context.Background()
		err := createLoadBalancerService(ctx, clients.Kubernetes, "elb", map[string]string{})
		Expect(err).NotTo(HaveOccurred())

		err = createLoadBalancerService(ctx, clients.Kubernetes, "ilb", map[string]string{
			"service.beta.kubernetes.io/azure-load-balancer-internal": "true",
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			return verifyServiceIsReady(ctx, clients.Kubernetes, "elb")
		}).Should(BeNil())

		Eventually(func() error {
			return verifyServiceIsReady(ctx, clients.Kubernetes, "ilb")
		}, 5*time.Minute, 1*time.Second).Should(BeNil())
	})
})

func createPVC(ctx context.Context, cli kubernetes.Interface) error {
	pvcStorage, err := resource.ParseQuantity("2Gi")
	if err != nil {
		return err
	}
	pvcName := fmt.Sprintf("%s-pvc", testPodName)
	storageClass := "managed-premium"
	_, err = cli.CoreV1().PersistentVolumeClaims(testNamespace).Create(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode("ReadWriteOnce"),
			},
			StorageClassName: to.StringPtr(storageClass),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: pvcStorage,
				},
			},
		},
	})
	return err
}

func createPodWithPVC(ctx context.Context, cli kubernetes.Interface) error {
	pvcName := fmt.Sprintf("%s-pvc", testPodName)
	volumeName := fmt.Sprintf("%s-vol", pvcName)
	_, err := cli.CoreV1().Pods(testNamespace).Create(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPodName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  testPodName,
					Image: testPodName,
					Command: []string{
						"/bin/dd",
						"if=/dev/urandom",
						fmt.Sprintf("of=/data/%s.bin", testNamespace),
						"bs=1M",
						"count=100",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/data",
							ReadOnly:  false,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	})
	return err
}

func verifyPodSucceeded(cli kubernetes.Interface) error {
	pod, err := cli.CoreV1().Pods(testNamespace).Get(testPodName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if pod.Status.Phase != corev1.PodSucceeded {
		return fmt.Errorf("Pod '%s' is not running: '%+v'", "busybox", pod.Status.Conditions)
	}
	return nil
}

func createLoadBalancerService(ctx context.Context, cli kubernetes.Interface, name string, annotations map[string]string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   testNamespace,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "port",
					Port: 8080,
				},
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}
	_, err := cli.CoreV1().Services(testNamespace).Create(svc)
	return err
}

func verifyServiceIsReady(ctx context.Context, cli kubernetes.Interface, svcName string) error {
	svc, err := cli.CoreV1().Services(testNamespace).Get(svcName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if len(svc.Status.LoadBalancer.Ingress) <= 0 {
			return fmt.Errorf("Load balancer service '%s' is not ready", svcName)
		}
	case corev1.ServiceTypeClusterIP:
		if net.ParseIP(svc.Spec.ClusterIP) == nil {
			return fmt.Errorf("ClusterIP service '%s' is not ready", svcName)
		}
	}
	return nil
}
