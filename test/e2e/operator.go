package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
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

func dumpEvents(ctx context.Context, namespace string) error {
	events, err := clients.Kubernetes.EventsV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, event := range events.Items {
		log.Debugf("%s. %s. %s", event.Action, event.Reason, event.Note)
	}
	return nil
}

var _ = Describe("ARO Operator - Internet checking", func() {
	var originalURLs []string
	BeforeEach(func() {
		// save the originalURLs
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			Skip("skipping tests as aro-operator is not deployed")
		}

		Expect(err).NotTo(HaveOccurred())
		originalURLs = co.Spec.InternetChecker.URLs
	})
	AfterEach(func() {
		// set the URLs back again
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = originalURLs
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(context.Background(), co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())
	})
	Specify("the InternetReachable default list should all be reachable", func() {
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(conditions.IsTrue(co.Status.Conditions, arov1alpha1.InternetReachableFromMaster)).To(BeTrue())
	})

	Specify("the InternetReachable default list should all be reachable from worker", func() {
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(conditions.IsTrue(co.Status.Conditions, arov1alpha1.InternetReachableFromWorker)).To(BeTrue())
	})

	Specify("custom invalid site shows not InternetReachable", func() {
		// set an unreachable URL
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = []string{"https://localhost:1234/shouldnotexist"}
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(context.Background(), co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		// confirm the conditions are correct
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, func() (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				log.Warn(err)
				return false, nil // swallow error
			}

			log.Debugf("ClusterStatus.Conditions %s", co.Status.Conditions)
			return conditions.IsFalse(co.Status.Conditions, arov1alpha1.InternetReachableFromMaster) &&
				conditions.IsFalse(co.Status.Conditions, arov1alpha1.InternetReachableFromWorker), nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - Geneva Logging", func() {
	Specify("genevalogging must be repaired if deployment deleted", func() {
		mdsdReady := func() (bool, error) {
			done, err := ready.CheckDaemonSetIsReady(context.Background(), clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging"), "mdsd")()
			if err != nil {
				log.Warn(err)
			}
			return done, nil // swallow error
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, mdsdReady)
		if err != nil {
			// TODO: Remove dump once reason for flakes is clear
			err := dumpEvents(context.Background(), "openshift-azure-logging")
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(err).NotTo(HaveOccurred())

		initial, err := updatedObjects(context.Background(), "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())

		// delete the mdsd daemonset
		err = clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete(context.Background(), "mdsd", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// wait for it to be fixed
		err = wait.PollImmediate(30*time.Second, 15*time.Minute, mdsdReady)
		if err != nil {
			// TODO: Remove dump once reason for flakes is clear
			err := dumpEvents(context.Background(), "openshift-azure-logging")
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(err).NotTo(HaveOccurred())

		// confirm that only one object was updated
		final, err := updatedObjects(context.Background(), "openshift-azure-logging")
		Expect(err).NotTo(HaveOccurred())
		if len(final)-len(initial) != 1 {
			log.Error("initial changes ", initial)
			log.Error("final changes ", final)
		}
		Expect(len(final) - len(initial)).To(Equal(1))
	})
})

var _ = Describe("ARO Operator - Cluster Monitoring ConfigMap", func() {
	Specify("cluster monitoring configmap should not have persistent volume config", func() {
		var cm *corev1.ConfigMap
		var err error
		configMapExists := func() (bool, error) {
			cm, err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(context.Background(), "cluster-monitoring-config", metav1.GetOptions{})
			if err != nil {
				return false, nil // swallow error
			}
			return true, nil
		}

		err = wait.PollImmediate(30*time.Second, 15*time.Minute, configMapExists)
		Expect(err).NotTo(HaveOccurred())

		var configData monitoring.Config
		configDataJSON, err := yaml.YAMLToJSON([]byte(cm.Data["config.yaml"]))
		Expect(err).NotTo(HaveOccurred())

		err = codec.NewDecoderBytes(configDataJSON, &codec.JsonHandle{}).Decode(&configData)
		if err != nil {
			log.Warn(err)
		}

		Expect(configData.PrometheusK8s.Retention).To(BeEmpty())
		Expect(configData.PrometheusK8s.VolumeClaimTemplate).To(BeNil())
		Expect(configData.AlertManagerMain.VolumeClaimTemplate).To(BeNil())

	})

	Specify("cluster monitoring configmap should be restored if deleted", func() {
		configMapExists := func() (bool, error) {
			_, err := clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(context.Background(), "cluster-monitoring-config", metav1.GetOptions{})
			if err != nil {
				return false, nil // swallow error
			}
			return true, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, configMapExists)
		Expect(err).NotTo(HaveOccurred())

		err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Delete(context.Background(), "cluster-monitoring-config", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = wait.PollImmediate(30*time.Second, 15*time.Minute, configMapExists)
		Expect(err).NotTo(HaveOccurred())

		_, err = clients.Kubernetes.CoreV1().ConfigMaps("openshift-monitoring").Get(context.Background(), "cluster-monitoring-config", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - RBAC", func() {
	Specify("system:aro-sre ClusterRole should be restored if deleted", func() {
		clusterRoleExists := func() (bool, error) {
			_, err := clients.Kubernetes.RbacV1().ClusterRoles().Get(context.Background(), "system:aro-sre", metav1.GetOptions{})
			if err != nil {
				return false, nil // swallow error
			}
			return true, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, clusterRoleExists)
		Expect(err).NotTo(HaveOccurred())

		err = clients.Kubernetes.RbacV1().ClusterRoles().Delete(context.Background(), "system:aro-sre", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = wait.PollImmediate(30*time.Second, 15*time.Minute, clusterRoleExists)
		Expect(err).NotTo(HaveOccurred())

		_, err = clients.Kubernetes.RbacV1().ClusterRoles().Get(context.Background(), "system:aro-sre", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - Conditions", func() {
	Specify("Cluster check conditions should not be failing", func() {
		// Save the last got conditions so that we can print them in the case of
		// the test failing
		var lastConditions []operatorv1.OperatorCondition

		clusterOperatorConditionsValid := func() (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			lastConditions = co.Status.Conditions

			valid := true
			for _, condition := range arov1alpha1.ClusterChecksTypes() {
				if !conditions.IsTrue(co.Status.Conditions, condition) {
					valid = false
				}
			}
			return valid, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, clusterOperatorConditionsValid)
		Expect(err).NotTo(HaveOccurred(), "last conditions: %v", lastConditions)
	})
})

var _ = Describe("ARO Operator - Azure Subnet Reconciler", func() {
	var vnetName, location, resourceGroup string
	var subnetsToReconcile map[string]*string
	var testnsg mgmtnetwork.SecurityGroup
	ctx := context.Background()

	const nsg = "e2e-nsg"

	// Gathers vnet name, resource group, location, and adds master/worker subnets to list to reconcile.
	gatherNetworkInfo := func() {
		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
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

	// Creates an empty NSG that gets assigned to master/worker subnets.
	createE2ENSG := func() {
		testnsg = mgmtnetwork.SecurityGroup{
			Location:                      &location,
			Name:                          to.StringPtr(nsg),
			Type:                          to.StringPtr("Microsoft.Network/networkSecurityGroups"),
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		}
		err := clients.NetworkSecurityGroups.CreateOrUpdateAndWait(ctx, resourceGroup, nsg, testnsg)
		Expect(err).NotTo(HaveOccurred())
		testnsg, err = clients.NetworkSecurityGroups.Get(ctx, resourceGroup, nsg, "")
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		gatherNetworkInfo()
		createE2ENSG()
	})
	AfterEach(func() {
		err := clients.NetworkSecurityGroups.DeleteAndWait(context.Background(), resourceGroup, nsg)
		if err != nil {
			log.Warn(err)
		}
	})
	It("must reconcile list of subnets when NSG is changed", func() {
		for subnet := range subnetsToReconcile {
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
			// Validate subnet reconciles to original NSG.
			err := wait.PollImmediate(30*time.Second, 10*time.Minute, func() (bool, error) {
				s, err := clients.Subnet.Get(ctx, resourceGroup, vnetName, subnet, "")
				if err != nil {
					return false, err
				}
				if *s.NetworkSecurityGroup.ID == *correctNSG {
					log.Infof("%s subnet's nsg matched expected value", subnet)
					return true, nil
				}
				log.Errorf("%s nsg: %s did not match expected value: %s", subnet, *s.NetworkSecurityGroup.ID, *correctNSG)
				return false, nil
			})
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

var _ = Describe("ARO Operator - MUO Deployment", func() {
	ctx := context.Background()

	Specify("MUO should be deployed by default", func() {
		muoIsDeployed := func() (bool, error) {
			pods, err := clients.Kubernetes.CoreV1().Pods("openshift-managed-upgrade-operator").List(ctx, metav1.ListOptions{
				LabelSelector: "name=managed-upgrade-operator",
			})
			if err != nil {
				return false, err
			}
			if len(pods.Items) != 1 {
				return false, fmt.Errorf("%d managed-upgrade-operator pods found", len(pods.Items))
			}
			b, err := clients.Kubernetes.CoreV1().Pods("openshift-managed-upgrade-operator").GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw(ctx)
			if err != nil {
				return false, err
			}

			// check for mandated FIPS
			return strings.Contains(string(b), `msg="FIPS crypto mandated: true"`), nil
		}

		err := wait.PollImmediate(30*time.Second, 10*time.Minute, muoIsDeployed)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - MHC Deployment", func() {
	Specify("MHC should be enabled and managed by default", func() {
		mhcIsDeployed := func() (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			mhcEnabled, _ := strconv.ParseBool(co.Spec.OperatorFlags.GetWithDefault("aro.machinehealthcheck.enabled", "false"))
			mhcManaged, _ := strconv.ParseBool(co.Spec.OperatorFlags.GetWithDefault("aro.machinehealthcheck.managed", "false"))

			if mhcEnabled && mhcManaged {
				return true, nil
			}
			return false, errors.New("mhc should be enabled and managed by default")
		}

		err := wait.PollImmediate(30*time.Second, 10*time.Minute, mhcIsDeployed)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - ImageConfig Reconciler", func() {
	const (
		imageconfigFlag  = "aro.imageconfig.enabled"
		optionalRegistry = "quay.io"
		timeout          = 5 * time.Minute
	)
	ctx := context.Background()

	var requiredRegistries []string
	var imageconfig *configv1.Image

	getLocationRegistries := func(acrDomain, acrDnsSuffix, location string) []string {
		regionalAcr := fmt.Sprintf("arosvc.%s.data.%s", location, acrDnsSuffix)
		requiredRegistries := []string{acrDomain, regionalAcr}
		return requiredRegistries
	}

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

	verifyLists := func(expectedAllowlist, expectedBlocklist []string) (bool, error) {
		// have to do this because using declaration assignment in following line results in pre-declared imageconfig var not being used
		var err error
		imageconfig, err = clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		allowList := imageconfig.Spec.RegistrySources.AllowedRegistries
		blockList := imageconfig.Spec.RegistrySources.BlockedRegistries

		return sliceEqual(allowList, expectedAllowlist) && sliceEqual(blockList, expectedBlocklist), nil
	}

	BeforeEach(func() {
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.OperatorFlags.GetSimpleBoolean(imageconfigFlag) {
			Skip("ImageConfig Controller is not enabled, skipping test")
		}

		imageconfig, err = clients.ConfigClient.ConfigV1().Images().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		requiredRegistries = getLocationRegistries(instance.Spec.ACRDomain, "azurecr.io", instance.Spec.Location)
	})

	AfterEach(func() {
		imageconfig.Spec.RegistrySources.AllowedRegistries = nil
		imageconfig.Spec.RegistrySources.BlockedRegistries = nil

		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			g.Expect(verifyLists(nil, nil)).To(BeTrue())
		}).WithTimeout(timeout).Should(Succeed())
	})

	It("should have empty allow and block lists by default", func() {
		allowList := imageconfig.Spec.RegistrySources.AllowedRegistries
		blockList := imageconfig.Spec.RegistrySources.BlockedRegistries

		Expect(allowList).To(BeEmpty())
		Expect(blockList).To(BeEmpty())
	})

	It("should add registries to the allow list", func() {

		expectedAllowlist := append(requiredRegistries, optionalRegistry)

		imageconfig.Spec.RegistrySources.AllowedRegistries = append(imageconfig.Spec.RegistrySources.AllowedRegistries, optionalRegistry)

		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			g.Expect(verifyLists(expectedAllowlist, nil)).To(BeTrue())
		}).WithTimeout(timeout).Should(Succeed())
	})

	It("should add optional registries to the block list and skip required registries", func() {
		expectedBlocklist := []string{optionalRegistry}

		imageconfig.Spec.RegistrySources.BlockedRegistries = append(imageconfig.Spec.RegistrySources.BlockedRegistries, optionalRegistry, requiredRegistries[0])

		_, err := clients.ConfigClient.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			g.Expect(verifyLists(nil, expectedBlocklist)).To(BeTrue())
		}).WithTimeout(timeout).Should(Succeed())
	})
})
