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
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	cov1Helpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	cpcController "github.com/Azure/ARO-RP/pkg/operator/controllers/cloudproviderconfig"
	imageController "github.com/Azure/ARO-RP/pkg/operator/controllers/imageconfig"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/monitoring"
	subnetController "github.com/Azure/ARO-RP/pkg/operator/controllers/subnets"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

func updatedObjects(ctx context.Context, nsFilter string) ([]string, error) {
	listFunc := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").List
	pods := ListK8sObjectWithRetry(
		ctx, listFunc, metav1.ListOptions{LabelSelector: "app=aro-operator-master"},
	)
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("%d aro-operator-master pods found", len(pods.Items))
	}
	body := GetK8sPodLogsWithRetry(
		ctx, "openshift-azure-operator", pods.Items[0].Name, corev1.PodLogOptions{},
	)

	rx := regexp.MustCompile(`msg="(Update|Create) ([-a-zA-Z/.]+)`)
	changes := rx.FindAllStringSubmatch(body, -1)
	result := make([]string, 0, len(changes))
	for _, change := range changes {
		if nsFilter == "" || strings.Contains(change[2], "/"+nsFilter+"/") {
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
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
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
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
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
		Eventually(mdsdIsReady).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		initial, err := updatedObjects(ctx, "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())

		By("deleting mdsd DaemonSet")
		DeleteK8sObjectWithRetry(
			ctx, clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete, "mdsd", metav1.DeleteOptions{},
		)

		By("checking that mdsd DaemonSet is ready")
		Eventually(mdsdIsReady).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

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
		getFunc := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get

		By("waiting for the ConfigMap to make sure it exists")
		cm = GetK8sObjectWithRetry(ctx, getFunc, "cluster-monitoring-config", metav1.GetOptions{})

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
		getFunc := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get
		deleteFunc := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Delete

		By("waiting for the ConfigMap to make sure it exists")
		GetK8sObjectWithRetry(ctx, getFunc, "cluster-monitoring-config", metav1.GetOptions{})

		By("deleting for the ConfigMap")
		DeleteK8sObjectWithRetry(ctx, deleteFunc, "cluster-monitoring-config", metav1.DeleteOptions{})

		By("waiting for the ConfigMap to make sure it was restored")
		GetK8sObjectWithRetry(ctx, getFunc, "cluster-monitoring-config", metav1.GetOptions{})
	})
})

var _ = Describe("ARO Operator - RBAC", func() {
	It("must restore system:aro-sre ClusterRole if deleted", func(ctx context.Context) {
		getFunc := clients.Kubernetes.RbacV1().ClusterRoles().Get
		deleteFunc := clients.Kubernetes.RbacV1().ClusterRoles().Delete

		By("waiting for the ClusterRole to make sure it exists")
		GetK8sObjectWithRetry(ctx, getFunc, "system:aro-sre", metav1.GetOptions{})

		By("deleting for the ClusterRole")
		DeleteK8sObjectWithRetry(ctx, deleteFunc, "system:aro-sre", metav1.DeleteOptions{})

		By("waiting for the ClusterRole to make sure it was restored")
		GetK8sObjectWithRetry(ctx, getFunc, "system:aro-sre", metav1.GetOptions{})
	})
})

var _ = Describe("ARO Operator - MachineHealthCheck", func() {
	const (
		mhcNamespace            = "openshift-machine-api"
		mhcName                 = "aro-machinehealthcheck"
		mhcRemediationAlertName = "mhc-remediation-alert"
	)

	getMachineHealthCheck := func(g Gomega, ctx context.Context) {
		_, err := clients.MachineAPI.MachineV1beta1().MachineHealthChecks(mhcNamespace).Get(ctx, mhcName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
	}

	getMachineHealthCheckRemediationAlertName := func(g Gomega, ctx context.Context) {
		_, err := clients.Monitoring.MonitoringV1().PrometheusRules(mhcNamespace).Get(ctx, mhcRemediationAlertName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
	}

	It("must be recreated if deleted", func(ctx context.Context) {
		By("deleting the machine health check")
		err := clients.MachineAPI.MachineV1beta1().MachineHealthChecks(mhcNamespace).Delete(ctx, mhcName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the machine health check to be restored")
		Eventually(getMachineHealthCheck).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})

	It("the alerting rule must recreated if deleted", func(ctx context.Context) {
		By("deleting the machine health remediation alert")
		err := clients.Monitoring.MonitoringV1().PrometheusRules(mhcNamespace).Delete(ctx, mhcRemediationAlertName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the machine health check remediation alert to be restored")
		Eventually(getMachineHealthCheckRemediationAlertName).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})

})

var _ = Describe("ARO Operator - Conditions", func() {
	const (
		timeout = 30 * time.Second
	)

	It("must have all the conditions on the cluster resource set to true", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			for _, condition := range arov1alpha1.ClusterChecksTypes() {
				g.Expect(conditions.IsTrue(co.Status.Conditions, condition)).To(BeTrue(), "Condition %s", condition)
			}
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})

	It("must have all the conditions on the cluster operator set to the expected values", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			co, err := clients.ConfigClient.ConfigV1().ClusterOperators().Get(ctx, "aro", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(cov1Helpers.IsStatusConditionTrue(co.Status.Conditions, configv1.OperatorAvailable)).To(BeTrue())
			g.Expect(cov1Helpers.IsStatusConditionFalse(co.Status.Conditions, configv1.OperatorProgressing)).To(BeTrue())
			g.Expect(cov1Helpers.IsStatusConditionFalse(co.Status.Conditions, configv1.OperatorDegraded)).To(BeTrue())
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).WithTimeout(timeout).Should(Succeed())
	})
})

func subnetReconciliationAnnotationExists(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}

	timestamp := annotations[subnetController.AnnotationTimestamp]
	_, err := time.Parse(time.RFC1123, timestamp)
	return err == nil
}

var _ = Describe("ARO Operator - Azure Subnet Reconciler", func() {
	var vnetName, location, resourceGroup string
	var subnetsToReconcile map[string]*string
	var testNSG mgmtnetwork.SecurityGroup

	const nsg = "e2e-nsg"
	const emptyMachineSet = "e2e-test-machineset"

	gatherNetworkInfo := func(ctx context.Context) {
		By("gathering vnet name, resource group, location, and adds master/worker subnets to list to reconcile")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		location = *oc.Location

		vnet, masterSubnet, err := apisubnet.Split((*oc.OpenShiftClusterProperties.MasterProfile.SubnetID))
		Expect(err).NotTo(HaveOccurred())

		_, workerSubnet, err := apisubnet.Split((*(*oc.OpenShiftClusterProperties.WorkerProfiles)[0].SubnetID))
		Expect(err).NotTo(HaveOccurred())

		subnetsToReconcile = map[string]*string{
			masterSubnet: to.StringPtr(""),
			workerSubnet: to.StringPtr(""),
		}

		r, err := azure.ParseResourceID(vnet)
		Expect(err).NotTo(HaveOccurred())
		resourceGroup = r.ResourceGroup
		vnetName = r.ResourceName

		// Store the existing NSGs for the cluster before the test runs, in order to ensure we clean up
		// after the test finishes, success or failure.
		// This is expensive but will prevent flakes.
		By("gathering existing subnet NSGs")
		for subnet := range subnetsToReconcile {
			subnetObject, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
			Expect(err).NotTo(HaveOccurred())

			subnetsToReconcile[subnet] = subnetObject.NetworkSecurityGroup.ID
		}
	}

	cleanUpSubnetNSGs := func(ctx context.Context) {
		By("cleaning up subnet NSGs")
		for subnet := range subnetsToReconcile {
			subnetObject, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
			Expect(err).NotTo(HaveOccurred())

			if subnetObject.NetworkSecurityGroup.ID != subnetsToReconcile[subnet] {
				subnetObject.NetworkSecurityGroup.ID = subnetsToReconcile[subnet]

				err = clients.Subnet.CreateOrUpdateAndWait(ctx, resourceGroup, vnetName, subnet, subnetObject)
				Expect(err).NotTo(HaveOccurred())
			}
		}
	}

	createE2ENSG := func(ctx context.Context) {
		By("creating an empty test NSG")
		testNSG = mgmtnetwork.SecurityGroup{
			Location:                      &location,
			Name:                          to.StringPtr(nsg),
			Type:                          to.StringPtr("Microsoft.Network/networkSecurityGroups"),
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		}
		err := clients.NetworkSecurityGroups.CreateOrUpdateAndWait(ctx, resourceGroup, nsg, testNSG)
		Expect(err).NotTo(HaveOccurred())

		By("getting the freshly created test NSG resource")
		testNSG, err = clients.NetworkSecurityGroups.Get(ctx, resourceGroup, nsg, "")
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func(ctx context.Context) {
		// TODO remove this when GA
		By("checking if preconfiguredNSG is enabled")
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if co.Spec.OperatorFlags[operator.AzureSubnetsNsgManaged] == operator.FlagFalse {
			Skip("preconfiguredNSG is enabled, skipping test")
		}
		By("preconfiguredNSG is disabled")

		gatherNetworkInfo(ctx)
		createE2ENSG(ctx)
	})
	AfterEach(func(ctx context.Context) {
		cleanUpSubnetNSGs(ctx)

		By("deleting test NSG")
		err := clients.NetworkSecurityGroups.DeleteAndWait(ctx, resourceGroup, nsg)
		if err != nil {
			log.Warn(err)
		}

		By("deleting test machineset if it still exists")
		err = clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").Delete(ctx, emptyMachineSet, metav1.DeleteOptions{})
		Expect(err).To(SatisfyAny(
			Not(HaveOccurred()),
			MatchError(kerrors.IsNotFound),
		))
	})
	It("must reconcile list of subnets when NSG is changed", func(ctx context.Context) {
		for subnet := range subnetsToReconcile {
			By(fmt.Sprintf("assigning test NSG to subnet %q", subnet))
			// Gets current subnet NSG and then updates it to testNSG.
			subnetObject, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
			Expect(err).NotTo(HaveOccurred())

			subnetObject.NetworkSecurityGroup = &testNSG

			err = clients.Subnet.CreateOrUpdateAndWait(ctx, resourceGroup, vnetName, subnet, subnetObject)
			Expect(err).NotTo(HaveOccurred())
		}

		By("creating an empty MachineSet to force a reconcile")
		Eventually(func(g Gomega, ctx context.Context) {
			machineSets, err := clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").List(ctx, metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(machineSets.Items).To(Not(BeEmpty()))

			newMachineSet := machineSets.Items[0].DeepCopy()
			newMachineSet.Status = machinev1beta1.MachineSetStatus{}
			newMachineSet.ObjectMeta = metav1.ObjectMeta{
				Name:        emptyMachineSet,
				Namespace:   "openshift-machine-api",
				Annotations: newMachineSet.ObjectMeta.Annotations,
				Labels:      newMachineSet.ObjectMeta.Labels,
			}
			newMachineSet.Name = emptyMachineSet
			newMachineSet.Spec.Replicas = to.Int32Ptr(0)

			_, err = clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").Create(ctx, newMachineSet, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		for subnet, correctNSG := range subnetsToReconcile {
			By(fmt.Sprintf("waiting for the subnet %q to be reconciled so it includes the original cluster NSG", subnet))
			Eventually(func(g Gomega, ctx context.Context) {
				s, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(*s.NetworkSecurityGroup.ID).To(Equal(*correctNSG))

				co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(co.Annotations).To(Satisfy(subnetReconciliationAnnotationExists))
			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
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
		pods := ListK8sObjectWithRetry(
			ctx, clients.Kubernetes.CoreV1().Pods(managedUpgradeOperatorNamespace).List, metav1.ListOptions{
				LabelSelector: "name=managed-upgrade-operator",
			})
		Expect(pods.Items).NotTo(BeEmpty())

		By("verifying that MUO has FIPS crypto mandated by reading logs")
		Eventually(func(g Gomega, ctx context.Context) {
			body := GetK8sPodLogsWithRetry(
				ctx, managedUpgradeOperatorNamespace, pods.Items[0].Name, corev1.PodLogOptions{},
			)

			g.Expect(body).To(ContainSubstring(`X:boringcrypto,strictfipsruntime`))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	}, SpecTimeout(2*time.Minute))

	It("must be restored if deleted", func(ctx context.Context) {
		deleteFunc := clients.Kubernetes.AppsV1().Deployments(managedUpgradeOperatorNamespace).Delete
		getFunc := clients.Kubernetes.AppsV1().Deployments(managedUpgradeOperatorNamespace).Get

		By("waiting for the MUO deployment to be ready")
		GetK8sObjectWithRetry(ctx, getFunc, managedUpgradeOperatorDeployment, metav1.GetOptions{})

		By("deleting the MUO deployment")
		DeleteK8sObjectWithRetry(ctx, deleteFunc, managedUpgradeOperatorDeployment, metav1.DeleteOptions{})

		By("waiting for the MUO deployment to be reconciled")
		GetK8sObjectWithRetry(ctx, getFunc, managedUpgradeOperatorDeployment, metav1.GetOptions{})
	}, SpecTimeout(2*time.Minute))
})

var _ = Describe("ARO Operator - ImageConfig Reconciler", func() {
	const (
		imageConfigFlag  = operator.ImageConfigEnabled
		optionalRegistry = "quay.io"
		timeout          = 5 * time.Minute
	)
	var requiredRegistries []string
	var imageConfig *configv1.Image

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
			// have to do this because using declaration assignment in following line results in pre-declared imageConfig var not being used
			var err error
			imageConfig, err = clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			By("comparing the actual allow and block lists with expected lists")
			g.Expect(sliceEqual(imageConfig.Spec.RegistrySources.AllowedRegistries, expectedAllowList)).To(BeTrue())
			g.Expect(sliceEqual(imageConfig.Spec.RegistrySources.BlockedRegistries, expectedBlockList)).To(BeTrue())
		}
	}

	BeforeEach(func(ctx context.Context) {
		By("checking whether Image config reconciliation is enabled in ARO operator config")
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.OperatorFlags.GetSimpleBoolean(imageConfigFlag) {
			Skip("ImageConfig Controller is not enabled, skipping test")
		}

		By("getting a list of required registries from the ARO operator config")
		requiredRegistries, err = imageController.GetCloudAwareRegistries(instance)
		Expect(err).NotTo(HaveOccurred())

		By("getting the Image config")
		imageConfig, err = clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx context.Context) {
		By("resetting Image config")
		imageConfig.Spec.RegistrySources.AllowedRegistries = nil
		imageConfig.Spec.RegistrySources.BlockedRegistries = nil

		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageConfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the Image config to be reset")
		Eventually(verifyLists(nil, nil)).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})

	It("must set empty allow and block lists in Image config by default", func() {
		allowList := imageConfig.Spec.RegistrySources.AllowedRegistries
		blockList := imageConfig.Spec.RegistrySources.BlockedRegistries

		By("checking that the allow and block lists are empty")
		Expect(allowList).To(BeEmpty())
		Expect(blockList).To(BeEmpty())
	})

	It("must add the ARO service registries to the allow list alongside the customer added registries", func(ctx context.Context) {
		By("adding the test registry to the allow list of the Image config")
		imageConfig.Spec.RegistrySources.AllowedRegistries = append(imageConfig.Spec.RegistrySources.AllowedRegistries, optionalRegistry)
		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageConfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that Image config eventually has ARO service registries and the test registry in the allow list")
		expectedAllowlist := append(requiredRegistries, optionalRegistry)
		Eventually(verifyLists(expectedAllowlist, nil)).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})

	It("must remove ARO service registries from the block lists, but keep customer added registries", func(ctx context.Context) {
		By("adding the test registry and one of the ARO service registry to the block list of the Image config")
		imageConfig.Spec.RegistrySources.BlockedRegistries = append(imageConfig.Spec.RegistrySources.BlockedRegistries, optionalRegistry, requiredRegistries[0])
		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageConfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("checking that Image config eventually doesn't include ARO service registries")
		expectedBlocklist := []string{optionalRegistry}
		Eventually(verifyLists(nil, expectedBlocklist)).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})
})

var _ = Describe("ARO Operator - dnsmasq", func() {
	const (
		timeout = 1 * time.Minute
		polling = 10 * time.Second
	)
	mcpName := "test-aro-custom-mcp"
	mcName := fmt.Sprintf("99-%s-aro-dns", mcpName)

	customMcp := mcv1.MachineConfigPool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machineconfiguration.openshift.io/v1",
			Kind:       "MachineConfigPool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mcpName,
		},
		Spec: mcv1.MachineConfigPoolSpec{},
	}

	getMachineConfigNames := func(g Gomega, ctx context.Context) []string {
		machineConfigs, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigs().List(ctx, metav1.ListOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		names := []string{}
		for _, mc := range machineConfigs.Items {
			names = append(names, mc.Name)
		}
		return names
	}

	BeforeEach(func(ctx context.Context) {
		By("Create custom MachineConfigPool")
		_, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigPools().Create(ctx, &customMcp, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("must handle the lifetime of the `99-${MCP}-custom-dns MachineConfig for every MachineConfigPool ${MCP}", Ordered, func() {
		It("should create an ARO DNS MachineConfig when creating a custom MachineConfigPool", func(ctx context.Context) {
			Eventually(func(g Gomega, ctx context.Context) []string {
				return getMachineConfigNames(g, ctx)
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(polling).
				Should(ContainElement(mcName))
		})

		It("should have the MachineConfig be owned by the Operator", func(ctx context.Context) {
			customMachineConfig, err := clients.MachineConfig.MachineconfigurationV1().MachineConfigs().Get(ctx, mcName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(customMachineConfig.ObjectMeta.OwnerReferences[0].Name).To(Equal(mcpName))
		})

		It("should delete the MachineConfig when deleting the custom MachineConfigPool", func(ctx context.Context) {
			err := clients.MachineConfig.MachineconfigurationV1().MachineConfigPools().Delete(ctx, mcpName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				machineConfigs := getMachineConfigNames(g, ctx)
				g.Expect(machineConfigs).NotTo(ContainElement(mcName))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(polling).Should(Succeed())
		})
	})
})

var _ = Describe("ARO Operator - Guardrails", func() {
	const (
		guardrailsEnabledFlag         = operator.GuardrailsEnabled
		guardrailsDeployManagedFlag   = operator.GuardrailsDeployManaged
		guardrailsNamespace           = "openshift-azure-guardrails"
		gkControllerManagerDeployment = "gatekeeper-controller-manager"
		gkAuditDeployment             = "gatekeeper-audit"
	)

	It("Controller Manager must be restored if deleted", func(ctx context.Context) {
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.OperatorFlags.GetSimpleBoolean(guardrailsEnabledFlag) ||
			!instance.Spec.OperatorFlags.GetSimpleBoolean(guardrailsDeployManagedFlag) {
			Skip("Guardrails Controller is not enabled, skipping test")
		}

		getFunc := clients.Kubernetes.AppsV1().Deployments(guardrailsNamespace).Get
		deleteFunc := clients.Kubernetes.AppsV1().Deployments(guardrailsNamespace).Delete

		By("waiting for the gatekeeper Controller Manager deployment to be ready")
		GetK8sObjectWithRetry(ctx, getFunc, gkControllerManagerDeployment, metav1.GetOptions{})

		By("deleting the gatekeeper Controller Manager deployment")
		DeleteK8sObjectWithRetry(ctx, deleteFunc, gkControllerManagerDeployment, metav1.DeleteOptions{})

		By("waiting for the gatekeeper Controller Manager deployment to be reconciled")
		GetK8sObjectWithRetry(ctx, getFunc, gkControllerManagerDeployment, metav1.GetOptions{})
	})

	It("Audit must be restored if deleted", func(ctx context.Context) {
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.OperatorFlags.GetSimpleBoolean(guardrailsEnabledFlag) ||
			!instance.Spec.OperatorFlags.GetSimpleBoolean(guardrailsDeployManagedFlag) {
			Skip("Guardrails Controller is not enabled, skipping test")
		}

		getFunc := clients.Kubernetes.AppsV1().Deployments(guardrailsNamespace).Get
		deleteFunc := clients.Kubernetes.AppsV1().Deployments(guardrailsNamespace).Delete

		By("waiting for the gatekeeper Audit deployment to be ready")
		GetK8sObjectWithRetry(ctx, getFunc, gkAuditDeployment, metav1.GetOptions{})

		By("deleting the gatekeeper Audit deployment")
		DeleteK8sObjectWithRetry(ctx, deleteFunc, gkAuditDeployment, metav1.DeleteOptions{})

		By("waiting for the gatekeeper Audit deployment to be reconciled")
		GetK8sObjectWithRetry(ctx, getFunc, gkAuditDeployment, metav1.GetOptions{})
	})

})

var _ = Describe("ARO Operator - Cloud Provider Config ConfigMap", func() {
	const (
		cpcControllerEnabled = operator.CloudProviderConfigEnabled
	)

	It("must have disableOutboundSNAT set to true", func(ctx context.Context) {
		By("checking whether CloudProviderConfig reconciliation is enabled in ARO operator config")
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.OperatorFlags.GetSimpleBoolean(cpcControllerEnabled) {
			Skip("CloudProviderConfig Controller is not enabled, skipping test")
		}

		By("waiting for the ConfigMap to make sure it exists")
		getFunc := clients.Kubernetes.CoreV1().ConfigMaps("openshift-config").Get
		cm := GetK8sObjectWithRetry(ctx, getFunc, "cloud-provider-config", metav1.GetOptions{})

		By("waiting for disableOutboundSNAT to be true")
		Eventually(func(g Gomega, ctx context.Context) {
			disableOutboundSNAT, err := cpcController.GetDisableOutboundSNAT(cm.Data["config"])
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(disableOutboundSNAT).To(BeTrue())
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})
})
