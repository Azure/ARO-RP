package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2022-09-04/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/test/util/project"
)

const (
	testPVCName = "e2e-test-claim"
)

var _ = Describe("Cluster", Serial, func() {
	var p project.Project

	BeforeEach(func(ctx context.Context) {
		By("creating a test namespace")
		testNamespace := fmt.Sprintf("test-e2e-%d", GinkgoParallelProcess())
		p = project.NewProject(clients.Kubernetes, clients.Project, testNamespace)
		err := p.Create(ctx)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

		By("verifying the namespace is ready")
		Eventually(func(ctx context.Context) error {
			return p.Verify(ctx)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeNil())

		DeferCleanup(func(ctx context.Context) {
			By("deleting a test namespace")
			err := p.Delete(ctx)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete test namespace")

			By("verifying the namespace is deleted")
			Eventually(func(ctx context.Context) error {
				return p.VerifyProjectIsDeleted(ctx)
			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeNil())
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

			ssName, err := createStatefulSet(ctx, clients.Kubernetes, p.Name, storageClass)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the stateful set is ready")
			Eventually(func(g Gomega, ctx context.Context) {
				s, err := clients.Kubernetes.AppsV1().StatefulSets(p.Name).Get(ctx, ssName, metav1.GetOptions{})
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

				mgmtSubnet, err := clients.Subnet.Get(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, "")
				Expect(err).NotTo(HaveOccurred())

				if mgmtSubnet.SubnetPropertiesFormat == nil {
					mgmtSubnet.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
				}

				if mgmtSubnet.SubnetPropertiesFormat.ServiceEndpoints == nil {
					mgmtSubnet.SubnetPropertiesFormat.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{}
				}

				// Check whether service endpoint is already there before trying to add
				// it; trying to add a duplicate results in an error
				for _, se := range *mgmtSubnet.ServiceEndpoints {
					if se.Service != nil && *se.Service == "Microsoft.Storage" {
						subnetAlreadyHasStorageEndpoint = true
						break
					}
				}

				if !subnetAlreadyHasStorageEndpoint {
					storageEndpoint := mgmtnetwork.ServiceEndpointPropertiesFormat{
						Service:   to.StringPtr("Microsoft.Storage"),
						Locations: &[]string{"*"},
					}

					*mgmtSubnet.ServiceEndpoints = append(*mgmtSubnet.ServiceEndpoints, storageEndpoint)

					err = clients.Subnet.CreateOrUpdateAndWait(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, mgmtSubnet)
					Expect(err).NotTo(HaveOccurred())
				}
			}

			// PUCM would be more reliable to check against,
			// but we cannot PUCM in prod, and dev clusters have ACLs set to allow
			By("checking the storage account vnet rules to verify that they include the cluster subnets")

			cluster, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Poke the ARO storageaccount controller to reconcile
			cluster.Spec.OperatorFlags["aro.storageaccounts.enabled"] = "false"
			cluster, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, cluster, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			cluster.Spec.OperatorFlags["aro.storageaccounts.enabled"] = "true"
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
					if rule.Action == storage.Allow && rule.VirtualNetworkResourceID != nil {
						nAclSubnets = append(nAclSubnets, strings.ToLower(*rule.VirtualNetworkResourceID))
					}
				}

				for _, subnet := range ocpSubnets {
					g.Expect(nAclSubnets).To(ContainElement(strings.ToLower(subnet)))
				}

			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

			By("creating stateful set")
			storageClass := "azurefile-csi"
			ssName, err := createStatefulSet(ctx, clients.Kubernetes, p.Name, storageClass)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the stateful set is ready")
			Eventually(func(g Gomega, ctx context.Context) {
				s, err := clients.Kubernetes.AppsV1().StatefulSets(p.Name).Get(ctx, ssName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(ready.StatefulSetIsReady(s)).To(BeTrue(), "expect stateful to be ready")

				pvcName := statefulSetPVCName(ssName, testPVCName, 0)
				pvc, err := clients.Kubernetes.CoreV1().PersistentVolumeClaims(p.Name).Get(ctx, pvcName, metav1.GetOptions{})
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

					mgmtSubnet, err := clients.Subnet.Get(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, "")
					Expect(err).NotTo(HaveOccurred())

					if mgmtSubnet.SubnetPropertiesFormat == nil {
						mgmtSubnet.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
					}

					mgmtSubnet.SubnetPropertiesFormat.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{}

					err = clients.Subnet.CreateOrUpdateAndWait(ctx, vnetResourceGroup, vnetR.ResourceName, subnetName, mgmtSubnet)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})

	})

	It("can create load balancer services", func(ctx context.Context) {
		By("creating an external load balancer service")
		err := createLoadBalancerService(ctx, clients.Kubernetes, "elb", p.Name, map[string]string{})
		Expect(err).NotTo(HaveOccurred())

		By("creating an internal load balancer service")
		err = createLoadBalancerService(ctx, clients.Kubernetes, "ilb", p.Name, map[string]string{
			"service.beta.kubernetes.io/azure-load-balancer-internal": "true",
		})
		Expect(err).NotTo(HaveOccurred())

		By("verifying the external load balancer service is ready")
		Eventually(func(ctx context.Context) bool {
			svc, err := clients.Kubernetes.CoreV1().Services(p.Name).Get(ctx, "elb", metav1.GetOptions{})
			if err != nil {
				return false
			}
			return ready.ServiceIsReady(svc)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeTrue())

		By("verifying the internal load balancer service is ready")
		Eventually(func(ctx context.Context) bool {
			svc, err := clients.Kubernetes.CoreV1().Services(p.Name).Get(ctx, "ilb", metav1.GetOptions{})
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
		err := createContainerFromInternalContainerRegistryImage(ctx, clients.Kubernetes, deployName, p.Name)
		Expect(err).NotTo(HaveOccurred())

		By("verifying the deployment is ready")
		Eventually(func(g Gomega, ctx context.Context) {
			s, err := clients.Kubernetes.AppsV1().Deployments(p.Name).Get(ctx, deployName, metav1.GetOptions{})
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

func createStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace, storageClass string) (string, error) {
	pvcStorage, err := resource.ParseQuantity("2Gi")
	if err != nil {
		return "", err
	}
	ssName := fmt.Sprintf("busybox-%s-%d", storageClass, GinkgoParallelProcess())

	_, err = cli.AppsV1().StatefulSets(namespace).Create(ctx, &appsv1.StatefulSet{
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
	return ssName, err
}

func statefulSetPVCName(ssName string, claimName string, ordinal int) string {
	return fmt.Sprintf("%s-%s-%d", claimName, ssName, ordinal)
}

func createLoadBalancerService(ctx context.Context, cli kubernetes.Interface, name, namespace string, annotations map[string]string) error {
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
	_, err := cli.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	return err
}

func createContainerFromInternalContainerRegistryImage(ctx context.Context, cli kubernetes.Interface, name, namespace string) error {
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
	_, err := cli.AppsV1().Deployments(namespace).Create(ctx, deploy, metav1.CreateOptions{})
	return err
}
