package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/util/ready"
	proj "github.com/Azure/ARO-RP/test/util/project"
)

const (
	testNamespace = "test-e2e"
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

	Specify("Can run a stateful set which is using Azure Disk storage", func() {
		ctx := context.Background()
		err := createStatefulSet(ctx, clients.Kubernetes)
		Expect(err).NotTo(HaveOccurred())

		err = wait.PollImmediate(10*time.Second, 15*time.Minute, ready.CheckStatefulSetIsReady(ctx, clients.Kubernetes.AppsV1().StatefulSets(testNamespace), "busybox"))
		Expect(err).NotTo(HaveOccurred())
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

func createStatefulSet(ctx context.Context, cli kubernetes.Interface) error {
	pvcStorage, err := resource.ParseQuantity("2Gi")
	if err != nil {
		return err
	}

	_, err = cli.AppsV1().StatefulSets(testNamespace).Create(context.Background(), &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "busybox",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "busybox"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "busybox"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "busybox",
							Command: []string{
								"/bin/sh",
								"-c",
								"while true; do sleep 1; done",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "busybox",
									MountPath: "/data",
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "busybox",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: to.StringPtr("managed-premium"),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: pvcStorage,
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	return err
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
