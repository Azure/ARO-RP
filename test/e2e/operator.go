package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	imageController "github.com/Azure/ARO-RP/pkg/operator/controllers/imageconfig"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func updatedObjects(ctx context.Context, nsfilter string) ([]string, error) {
	pods, err := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").List(ctx, metav1.ListOptions{
		LabelSelector: "app=aro-operator-master",
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("%d aro-operator-master pods found", len(pods.Items))
	}
	b, err := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw(ctx)
	if err != nil {
		return nil, err
	}

	rx := regexp.MustCompile(`msg="(Update|Create) ([-a-zA-Z/.]+)`)
	changes := rx.FindAllStringSubmatch(string(b), -1)
	result := make([]string, 0, len(changes))
	for _, change := range changes {
		if nsfilter == "" || strings.Contains(change[2], "/"+nsfilter+"/") {
			result = append(result, change[1]+" "+change[2])
		}
	}

	return result, nil
}

var _ = Describe("ARO Operator - Internet checking", func() {
	var originalURLs []string
	BeforeEach(func(ctx context.Context) {
		By("saving the original URLs")
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			Skip("skipping tests as aro-operator is not deployed")
		}

		Expect(err).NotTo(HaveOccurred())
		originalURLs = co.Spec.InternetChecker.URLs
	})
	AfterEach(func(ctx context.Context) {
		By("restoring the original URLs")
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = originalURLs
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the original conditions")
		Eventually(func(g Gomega, ctx context.Context) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(conditions.IsTrue(co.Status.Conditions, arov1alpha1.InternetReachableFromMaster)).To(BeTrue())
			g.Expect(conditions.IsTrue(co.Status.Conditions, arov1alpha1.InternetReachableFromWorker)).To(BeTrue())
		}).WithContext(ctx).Should(Succeed())
	})

	It("sets InternetReachableFromMaster and InternetReachableFromWorker to false when URL is not reachable", func(ctx context.Context) {
		By("setting a deliberately unreachable URL")
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = []string{"https://localhost:1234/shouldnotexist"}
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the expected conditions to be set")
		Eventually(func(g Gomega, ctx context.Context) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(conditions.IsFalse(co.Status.Conditions, arov1alpha1.InternetReachableFromMaster)).To(BeTrue())
			g.Expect(conditions.IsFalse(co.Status.Conditions, arov1alpha1.InternetReachableFromWorker)).To(BeTrue())
		}).WithContext(ctx).Should(Succeed())
	})
})

var _ = Describe("ARO Operator - Geneva Logging", func() {
	It("must be repaired if DaemonSet deleted", func(ctx context.Context) {
		mdsdIsReady := func(g Gomega, ctx context.Context) {
			done, err := ready.CheckDaemonSetIsReady(ctx, clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging"), "mdsd")()

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(done).To(BeTrue())
		}

		By("checking that mdsd DaemonSet is ready before the test")
		Eventually(mdsdIsReady).WithContext(ctx).Should(Succeed())

		initial, err := updatedObjects(ctx, "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())

		By("deleting mdsd DaemonSet")
		err = clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete(ctx, "mdsd", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that mdsd DaemonSet is ready")
		Eventually(mdsdIsReady).WithContext(ctx).Should(Succeed())

		By("confirming that only one object was updated")
		final, err := updatedObjects(ctx, "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())
		if len(final)-len(initial) != 1 {
			log.Error("initial changes ", initial)
			log.Error("final changes ", final)
		}
		Expect(len(final) - len(initial)).To(Equal(1))
	})
})

var _ = Describe("ARO Operator - Cluster Monitoring ConfigMap", func() {
	It("must not have persistent volume set", func(ctx context.Context) {
		var cm *corev1.ConfigMap
		configMapExists := func(g Gomega, ctx context.Context) {
			var err error
			cm, err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(ctx, "cluster-monitoring-config", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
		}

		By("waiting for the ConfigMap to make sure it exists")
		Eventually(configMapExists).WithContext(ctx).Should(Succeed())

		By("unmarshalling the config from the ConfigMap data")
		var configData monitoring.Config
		configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
		Expect(err).NotTo(HaveOccurred())

		err = codec.NewDecoderBytes(configDataJSON, &codec.JsonHandle{}).Decode(&configData)
		if err != nil {
			log.Warn(err)
		}

		By("checking config correctness")
		Expect(configData.PrometheusK8s.Retention).To(BeEmpty())
		Expect(configData.PrometheusK8s.VolumeClaimTemplate).To(BeNil())
		Expect(configData.AlertManagerMain.VolumeClaimTemplate).To(BeNil())
	})

	It("must be restored if deleted", func(ctx context.Context) {
		configMapExists := func(g Gomega, ctx context.Context) {
			_, err := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(ctx, "cluster-monitoring-config", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
		}

		By("waiting for the ConfigMap to make sure it exists")
		Eventually(configMapExists).WithContext(ctx).Should(Succeed())

		By("deleting for the ConfigMap")
		err := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Delete(ctx, "cluster-monitoring-config", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the ConfigMap to make sure it was restored")
		Eventually(configMapExists).WithContext(ctx).Should(Succeed())
	})
})

var _ = Describe("ARO Operator - RBAC", func() {
	It("must restore system:aro-sre ClusterRole if deleted", func(ctx context.Context) {
		clusterRoleExists := func(g Gomega, ctx context.Context) {
			_, err := clients.Kubernetes.RbacV1().ClusterRoles().Get(ctx, "system:aro-sre", metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
		}

		By("waiting for the ClusterRole to make sure it exists")
		Eventually(clusterRoleExists).WithContext(ctx).Should(Succeed())

		By("deleting for the ClusterRole")
		err := clients.Kubernetes.RbacV1().ClusterRoles().Delete(ctx, "system:aro-sre", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the ClusterRole to make sure it was restored")
		Eventually(clusterRoleExists).WithContext(ctx).Should(Succeed())
	})
})

var _ = Describe("ARO Operator - Conditions", func() {
	It("must have all the conditions set to true", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			for _, condition := range arov1alpha1.ClusterChecksTypes() {
				g.Expect(conditions.IsTrue(co.Status.Conditions, condition)).To(BeTrue(), "Condition %s", condition)
			}
		}).WithContext(ctx).Should(Succeed())
	})
})

var _ = Describe("ARO Operator - Azure Subnet Reconciler", func() {
	var vnetName, location, resourceGroup string
	var subnetsToReconcile map[string]*string
	var testnsg mgmtnetwork.SecurityGroup

	const nsg = "e2e-nsg"

	gatherNetworkInfo := func(ctx context.Context) {
		By("gathering vnet name, resource group, location, and adds master/worker subnets to list to reconcile")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		location = *oc.Location

		vnet, masterSubnet, err := subnet.Split(*oc.OpenShiftClusterProperties.MasterProfile.SubnetID)
		Expect(err).NotTo(HaveOccurred())
		_, workerSubnet, err := subnet.Split((*(*oc.OpenShiftClusterProperties.WorkerProfiles)[0].SubnetID))
		Expect(err).NotTo(HaveOccurred())

		subnetsToReconcile = map[string]*string{
			masterSubnet: to.StringPtr(""),
			workerSubnet: to.StringPtr(""),
		}

		r, err := azure.ParseResourceID(vnet)
		Expect(err).NotTo(HaveOccurred())
		resourceGroup = r.ResourceGroup
		vnetName = r.ResourceName
	}

	createE2ENSG := func(ctx context.Context) {
		By("creating an empty test NSG")
		testnsg = mgmtnetwork.SecurityGroup{
			Location:                      &location,
			Name:                          to.StringPtr(nsg),
			Type:                          to.StringPtr("Microsoft.Network/networkSecurityGroups"),
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		}
		err := clients.NetworkSecurityGroups.CreateOrUpdateAndWait(ctx, resourceGroup, nsg, testnsg)
		Expect(err).NotTo(HaveOccurred())

		By("getting the freshly created test NSG resource")
		testnsg, err = clients.NetworkSecurityGroups.Get(ctx, resourceGroup, nsg, "")
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func(ctx context.Context) {
		gatherNetworkInfo(ctx)
		createE2ENSG(ctx)
	})
	AfterEach(func(ctx context.Context) {
		By("deleting test NSG")
		err := clients.NetworkSecurityGroups.DeleteAndWait(ctx, resourceGroup, nsg)
		if err != nil {
			log.Warn(err)
		}
	})
	It("must reconcile list of subnets when NSG is changed", func(ctx context.Context) {
		for subnet := range subnetsToReconcile {
			By(fmt.Sprintf("assigning test NSG to subnet %q", subnet))
			// Gets current subnet NSG and then updates it to testnsg.
			subnetObject, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
			Expect(err).NotTo(HaveOccurred())
			// Updates the value to the original NSG in our subnetsToReconcile map
			subnetsToReconcile[subnet] = subnetObject.NetworkSecurityGroup.ID
			subnetObject.NetworkSecurityGroup = &testnsg
			err = clients.Subnet.CreateOrUpdateAndWait(ctx, resourceGroup, vnetName, subnet, subnetObject)
			Expect(err).NotTo(HaveOccurred())
		}

		for subnet, correctNSG := range subnetsToReconcile {
			By(fmt.Sprintf("waiting for the subnet %q to be reconciled so it includes the original cluster NSG", subnet))
			Eventually(func(g Gomega, ctx context.Context) {
				s, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(*s.NetworkSecurityGroup.ID).To(Equal(*correctNSG))
			}).WithContext(ctx).Should(Succeed())
		}
	})
})

var _ = Describe("ARO Operator - MUO Deployment", func() {
	const (
		managedUpgradeOperatorNamespace  = "openshift-managed-upgrade-operator"
		managedUpgradeOperatorDeployment = "managed-upgrade-operator"
	)

	It("must be deployed by default with FIPS crypto mandated", func(ctx context.Context) {
		By("getting MUO pods")
		pods, err := clients.Kubernetes.CoreV1().Pods(managedUpgradeOperatorNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "name=managed-upgrade-operator",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())

		By("verifying that MUO has FIPS crypto mandated by reading logs")
		Eventually(func(g Gomega, ctx context.Context) {
			b, err := clients.Kubernetes.CoreV1().Pods(managedUpgradeOperatorNamespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw(ctx)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(string(b)).To(ContainSubstring(`msg="FIPS crypto mandated: true"`))
		}).WithContext(ctx).Should(Succeed())
	})

	It("must be restored if deleted", func(ctx context.Context) {
		deleteMUODeployment := func(ctx context.Context) error {
			return clients.Kubernetes.
				AppsV1().
				Deployments(managedUpgradeOperatorNamespace).
				Delete(ctx, managedUpgradeOperatorDeployment, metav1.DeleteOptions{})
		}

		muoDeploymentExists := func(g Gomega, ctx context.Context) {
			_, err := clients.Kubernetes.
				AppsV1().
				Deployments(managedUpgradeOperatorNamespace).
				Get(ctx, managedUpgradeOperatorDeployment, metav1.GetOptions{})

			g.Expect(err).ToNot(HaveOccurred())
		}

		By("waiting for the MUO deployment to be ready")
		Eventually(muoDeploymentExists).WithContext(ctx).Should(Succeed())

		By("deleting the MUO deployment")
		Expect(deleteMUODeployment(ctx)).Should(Succeed())

		By("waiting for the MUO deployment to be reconciled")
		Eventually(muoDeploymentExists).WithContext(ctx).Should(Succeed())
	})
})

var _ = Describe("ARO Operator - ImageConfig Reconciler", func() {
	const (
		imageconfigFlag  = "aro.imageconfig.enabled"
		optionalRegistry = "quay.io"
		timeout          = 5 * time.Minute
	)
	var requiredRegistries []string
	var imageconfig *configv1.Image

	sliceEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		sort.Strings(a)
		sort.Strings(b)

		for idx, entry := range b {
			if a[idx] != entry {
				return false
			}
		}
		return true
	}

	// verifyLists generates a closure to be called inside of Eventually.
	// The closure will be called multiple times until it is
	// eventually meets expectations or exceeds timeout.
	verifyLists := func(expectedAllowList, expectedBlockList []string) func(g Gomega, ctx context.Context) {
		return func(g Gomega, ctx context.Context) {
			By("getting the actual Image config state")
			// have to do this because using declaration assignment in following line results in pre-declared imageconfig var not being used
			var err error
			imageconfig, err = clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			By("comparing the actual allow and block lists with expected lists")
			g.Expect(sliceEqual(imageconfig.Spec.RegistrySources.AllowedRegistries, expectedAllowList)).To(BeTrue())
			g.Expect(sliceEqual(imageconfig.Spec.RegistrySources.BlockedRegistries, expectedBlockList)).To(BeTrue())
		}
	}

	BeforeEach(func(ctx context.Context) {
		By("checking whether Image config reconciliation is enabled in ARO operator config")
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.OperatorFlags.GetSimpleBoolean(imageconfigFlag) {
			Skip("ImageConfig Controller is not enabled, skipping test")
		}

		By("getting a list of required registries from the ARO operator config")
		requiredRegistries, err = imageController.GetCloudAwareRegistries(instance)
		Expect(err).NotTo(HaveOccurred())

		By("getting the Image config")
		imageconfig, err = clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx context.Context) {
		By("resetting Image config")
		imageconfig.Spec.RegistrySources.AllowedRegistries = nil
		imageconfig.Spec.RegistrySources.BlockedRegistries = nil

		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the Image config to be reset")
		Eventually(verifyLists(nil, nil)).WithContext(ctx).Should(Succeed())
	})

	It("must set empty allow and block lists in Image config by default", func() {
		allowList := imageconfig.Spec.RegistrySources.AllowedRegistries
		blockList := imageconfig.Spec.RegistrySources.BlockedRegistries

		By("checking that the allow and block lists are empty")
		Expect(allowList).To(BeEmpty())
		Expect(blockList).To(BeEmpty())
	})

	It("must add the ARO service registries to the allow list alongside the customer added registries", func(ctx context.Context) {
		By("adding the test registry to the allow list of the Image config")
		imageconfig.Spec.RegistrySources.AllowedRegistries = append(imageconfig.Spec.RegistrySources.AllowedRegistries, optionalRegistry)
		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that Image config eventually has ARO service registries and the test registry in the allow list")
		expectedAllowlist := append(requiredRegistries, optionalRegistry)
		Eventually(verifyLists(expectedAllowlist, nil)).WithContext(ctx).Should(Succeed())
	})

	It("must remove ARO service registries from the block lists, but keep customer added registries", func(ctx context.Context) {
		By("adding the test registry and one of the ARO service registry to the block list of the Image config")
		imageconfig.Spec.RegistrySources.BlockedRegistries = append(imageconfig.Spec.RegistrySources.BlockedRegistries, optionalRegistry, requiredRegistries[0])
		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that Image config eventually doesn't include ARO service registries")
		expectedBlocklist := []string{optionalRegistry}
		Eventually(verifyLists(nil, expectedBlocklist)).WithContext(ctx).Should(Succeed())
	})
})
