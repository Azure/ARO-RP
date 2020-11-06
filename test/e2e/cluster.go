package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ready "github.com/Azure/ARO-RP/pkg/util/ready"
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
		err := project.Create(context.Background())
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

		Eventually(func() error {
			return project.Verify(context.Background())
		}).Should(BeNil())
	})

	var _ = AfterEach(func() {
		err := project.Delete(context.Background())
		Expect(err).NotTo(HaveOccurred(), "Failed to delete test namespace")

		Eventually(func() error {
			return project.VerifyProjectIsDeleted(context.Background())
		}, 5*time.Minute, 10*time.Second).Should(BeNil())
	})

	Specify("Can run a pod which is using Azure Disk storage", func() {
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

		Eventually(func() bool {
			svc, err := clients.Kubernetes.CoreV1().Services(testNamespace).Get(context.Background(), "elb", metav1.GetOptions{})
			if err != nil {
				return false
			}
			return ready.ServiceIsReady(svc)
		}, 5*time.Minute, 10*time.Second).Should(BeTrue())

		Eventually(func() bool {
			svc, err := clients.Kubernetes.CoreV1().Services(testNamespace).Get(context.Background(), "ilb", metav1.GetOptions{})
			if err != nil {
				return false
			}
			return ready.ServiceIsReady(svc)
		}, 5*time.Minute, 10*time.Second).Should(BeTrue())
	})
})

func createPVC(ctx context.Context, cli kubernetes.Interface) error {
	pvcStorage, err := resource.ParseQuantity("2Gi")
	if err != nil {
		return err
	}
	pvcName := fmt.Sprintf("%s-pvc", testPodName)
	storageClass := "managed-premium"
	_, err = cli.CoreV1().PersistentVolumeClaims(testNamespace).Create(context.Background(), &corev1.PersistentVolumeClaim{
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
	}, metav1.CreateOptions{})
	return err
}

func createPodWithPVC(ctx context.Context, cli kubernetes.Interface) error {
	pvcName := fmt.Sprintf("%s-pvc", testPodName)
	volumeName := fmt.Sprintf("%s-vol", pvcName)
	_, err := cli.CoreV1().Pods(testNamespace).Create(context.Background(), &corev1.Pod{
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
	}, metav1.CreateOptions{})
	return err
}

func verifyPodSucceeded(cli kubernetes.Interface) error {
	pod, err := cli.CoreV1().Pods(testNamespace).Get(context.Background(), testPodName, metav1.GetOptions{})
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
	_, err := cli.CoreV1().Services(testNamespace).Create(context.Background(), svc, metav1.CreateOptions{})
	return err
}
