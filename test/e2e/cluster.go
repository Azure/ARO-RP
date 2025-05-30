package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	testPVCName = "e2e-test-claim"
)

var _ = Describe("Cluster", Serial, func() {
	var project Project

	BeforeEach(func(ctx context.Context) {
		By("creating a test namespace")
		testNamespace := fmt.Sprintf("test-e2e-%d", GinkgoParallelProcess())
		project = BuildNewProject(ctx, clients.Kubernetes, clients.Project, testNamespace)

		By("verifying the namespace is ready")
		Eventually(func(ctx context.Context) error {
			return project.VerifyProjectIsReady(ctx)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		DeferCleanup(func(ctx context.Context) {
			By("deleting the test project")
			project.CleanUp(ctx)
		})
	})

	Context("can run a stateful set", func() {
		It("which is using Azure Disk storage", func(ctx context.Context) {
			By("creating stateful set")
			oc, _ := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
			installVersion, _ := version.ParseVersion(*oc.ClusterProfile.Version)

			storageClass := "managed-csi"

			if installVersion.Lt(version.NewVersion(4, 11)) {
				storageClass = "managed-premium"
			}

			ssName := createStatefulSet(ctx, clients.Kubernetes, project.Name, storageClass)

			By("verifying the stateful set is ready")
			Eventually(func(g Gomega, ctx context.Context) {
				s, err := clients.Kubernetes.AppsV1().StatefulSets(project.Name).Get(ctx, ssName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ready.StatefulSetIsReady(s)).To(BeTrue(), "expect stateful to be ready")
				GinkgoWriter.Println(s)
			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
		})

		// TODO: This test is marked as Pending because CI clusters are FIPS-enabled, and Azure File storage
		// doesn't work with FIPS-enabled clusters: https://learn.microsoft.com/en-us/azure/openshift/howto-enable-fips-openshift#support-for-fips-cryptography
		//
		// We should enable this test when/if FIPS becomes toggleable post-install in the future.
		It("which is using the default Azure File storage class backed by the cluster storage account", Pending, func(ctx context.Context) {
			By("adding the Microsoft.Storage service endpoint to each cluster subnet (if needed)")

			oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
			Expect(err).NotTo(HaveOccurred())
			ocpSubnets := clusterSubnets(oc)
			subnetAlreadyHasStorageEndpoint := false

			for _, s := range ocpSubnets {
				vnetID, subnetName, err := apisubnet.Split(s)
				Expect(err).NotTo(HaveOccurred())

				vnetR, err := azure.ParseResourceID(vnetID)
				Expect(err).NotTo(HaveOccurred())

				resp, err := clients.Subnet.Get(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, nil)
				Expect(err).NotTo(HaveOccurred())
				subnet := resp.Subnet

				if subnet.Properties == nil {
					subnet.Properties = &armnetwork.SubnetPropertiesFormat{}
				}

				if subnet.Properties.ServiceEndpoints == nil {
					subnet.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{}
				}

				// Check whether service endpoint is already there before trying to add
				// it; trying to add a duplicate results in an error
				for _, se := range subnet.Properties.ServiceEndpoints {
					if se.Service != nil && *se.Service == "Microsoft.Storage" {
						subnetAlreadyHasStorageEndpoint = true
						break
					}
				}

				if !subnetAlreadyHasStorageEndpoint {
					storageEndpoint := armnetwork.ServiceEndpointPropertiesFormat{
						Service:   to.StringPtr("Microsoft.Storage"),
						Locations: []*string{to.StringPtr("*")},
					}

					subnet.Properties.ServiceEndpoints = append(subnet.Properties.ServiceEndpoints, &storageEndpoint)

					err = clients.Subnet.CreateOrUpdateAndWait(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, subnet, nil)
					Expect(err).NotTo(HaveOccurred())
				}
			}

			// PUCM would be more reliable to check against,
			// but we cannot PUCM in prod, and dev clusters have ACLs set to allow
			By("checking the storage account vnet rules to verify that they include the cluster subnets")

			cluster, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Poke the ARO storage account controller to reconcile
			cluster.Spec.OperatorFlags[operator.StorageAccountsEnabled] = operator.FlagFalse
			cluster, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, cluster, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			cluster.Spec.OperatorFlags[operator.StorageAccountsEnabled] = operator.FlagTrue
			cluster, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, cluster, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			rgName := stringutils.LastTokenByte(cluster.Spec.ClusterResourceGroupID, '/')

			// only checking the cluster storage account
			Eventually(func(g Gomega, ctx context.Context) {
				account, err := clients.Storage.GetProperties(ctx, rgName, "cluster"+cluster.Spec.StorageSuffix, "")
				g.Expect(err).NotTo(HaveOccurred())

				nAclSubnets := []string{}
				g.Expect(account.AccountProperties).NotTo(BeNil())
				g.Expect(account.NetworkRuleSet).NotTo(BeNil())
				g.Expect(account.NetworkRuleSet.VirtualNetworkRules).NotTo(BeNil())

				for _, rule := range *account.NetworkRuleSet.VirtualNetworkRules {
					if rule.Action == storage.ActionAllow && rule.VirtualNetworkResourceID != nil {
						nAclSubnets = append(nAclSubnets, strings.ToLower(*rule.VirtualNetworkResourceID))
					}
				}

				for _, subnet := range ocpSubnets {
					g.Expect(nAclSubnets).To(ContainElement(strings.ToLower(subnet)))
				}

			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

			By("creating stateful set")
			storageClass := "azurefile-csi"
			ssName := createStatefulSet(ctx, clients.Kubernetes, project.Name, storageClass)

			By("verifying the stateful set is ready")
			Eventually(func(g Gomega, ctx context.Context) {
				s, err := clients.Kubernetes.AppsV1().StatefulSets(project.Name).Get(ctx, ssName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ready.StatefulSetIsReady(s)).To(BeTrue(), "expect stateful to be ready")

				pvcName := statefulSetPVCName(ssName, testPVCName, 0)
				pvc, err := clients.Kubernetes.CoreV1().PersistentVolumeClaims(project.Name).Get(ctx, pvcName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Println(pvc)
			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

			// The cluster subnets should always have endpoints in CI since CI doesn't have the gateway, but being safe
			By("cleaning up the cluster subnets (i.e. removing service endpoints if appropriate)")
			if !subnetAlreadyHasStorageEndpoint {
				for _, s := range ocpSubnets {
					vnetID, subnetName, err := apisubnet.Split(s)
					Expect(err).NotTo(HaveOccurred())

					vnetR, err := azure.ParseResourceID(vnetID)
					Expect(err).NotTo(HaveOccurred())

					resp, err := clients.Subnet.Get(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, nil)
					Expect(err).NotTo(HaveOccurred())
					subnet := resp.Subnet

					if subnet.Properties == nil {
						subnet.Properties = &armnetwork.SubnetPropertiesFormat{}
					}

					subnet.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{}

					err = clients.Subnet.CreateOrUpdateAndWait(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, subnet, nil)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})

	})

	It("can create load balancer services", func(ctx context.Context) {
		By("creating an external load balancer service")
		createLoadBalancerService(ctx, clients.Kubernetes, "elb", project.Name, map[string]string{})

		By("creating an internal load balancer service")
		createLoadBalancerService(ctx, clients.Kubernetes, "ilb", project.Name, map[string]string{
			"service.beta.kubernetes.io/azure-load-balancer-internal": "true",
		})

		By("verifying the external load balancer service is ready")
		Eventually(func(ctx context.Context) bool {
			svc, err := clients.Kubernetes.CoreV1().Services(project.Name).Get(ctx, "elb", metav1.GetOptions{})
			if err != nil {
				return false
			}
			return ready.ServiceIsReady(svc)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeTrue())

		By("verifying the internal load balancer service is ready")
		Eventually(func(ctx context.Context) bool {
			svc, err := clients.Kubernetes.CoreV1().Services(project.Name).Get(ctx, "ilb", metav1.GetOptions{})
			if err != nil {
				return false
			}
			return ready.ServiceIsReady(svc)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeTrue())
	})

	// mainly we want to test the gateway/egress functionality - this request for the image will travel from
	// node > gateway > storage account of the registry.
	It("can access and use the internal container registry", func(ctx context.Context) {
		deployName := "internal-registry-deploy"

		By("creating a test deployment from an internal container registry")
		createContainerFromInternalContainerRegistryImage(ctx, clients.Kubernetes, deployName, project.Name)

		By("verifying the deployment is ready")
		Eventually(func(g Gomega, ctx context.Context) {
			s, err := clients.Kubernetes.AppsV1().Deployments(project.Name).Get(ctx, deployName, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(ready.DeploymentIsReady(s)).To(BeTrue(), "expect stateful to be ready")
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})
})

// clusterSubnets returns a slice containing all of the cluster subnets' resource IDs
func clusterSubnets(oc redhatopenshift.OpenShiftCluster) []string {
	subnetMap := map[string]struct{}{}
	subnetMap[*oc.OpenShiftClusterProperties.MasterProfile.SubnetID] = struct{}{}

	// TODO: change to workerProfileStatuses when we bump the API to 20230904 stable
	for _, p := range *oc.OpenShiftClusterProperties.WorkerProfiles {
		s := strings.ToLower(*p.SubnetID)
		subnetMap[s] = struct{}{}
	}

	subnets := []string{}

	for subnet := range subnetMap {
		subnets = append(subnets, subnet)
	}

	return subnets
}

func createStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace, storageClass string) string {
	quantity := "2Gi"
	pvcStorage, err := resource.ParseQuantity(quantity)
	if err != nil {
		message := fmt.Sprintf("Could not parse %v when creating a stateful set.", quantity)
		Fail(message)
	}

	ssName := fmt.Sprintf("busybox-%s-%d", storageClass, GinkgoParallelProcess())

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: ssName,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": ssName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": ssName},
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
									Name:      testPVCName,
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
						Name: testPVCName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: to.StringPtr(storageClass),
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: pvcStorage,
							},
						},
					},
				},
			},
		},
	}

	_ = CreateK8sObjectWithRetry(ctx, cli.AppsV1().StatefulSets(namespace).Create, ss, metav1.CreateOptions{})
	return ssName
}

func statefulSetPVCName(ssName string, claimName string, ordinal int) string {
	return fmt.Sprintf("%s-%s-%d", claimName, ssName, ordinal)
}

func createLoadBalancerService(ctx context.Context, cli kubernetes.Interface, name, namespace string, annotations map[string]string) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
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
	CreateK8sObjectWithRetry(ctx, cli.CoreV1().Services(namespace).Create, svc, metav1.CreateOptions{})
}

func createContainerFromInternalContainerRegistryImage(ctx context.Context, cli kubernetes.Interface, name, namespace string) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: to.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cli",
							Image: "image-registry.openshift-image-registry.svc:5000/openshift/cli",
							Command: []string{
								"/bin/sh",
								"-c",
								"while true; do sleep 1; done",
							},
						},
					},
				},
			},
		},
	}
	CreateK8sObjectWithRetry(ctx, cli.AppsV1().Deployments(namespace).Create, deploy, metav1.CreateOptions{})
}
