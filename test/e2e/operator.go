package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
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
		clusterOperatorConditionsValid := func() (bool, error) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(context.Background(), "cluster", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			valid := true
			for _, condition := range arov1alpha1.ClusterChecksTypes() {
				if !conditions.IsTrue(co.Status.Conditions, condition) {
					valid = false
				}
			}
			return valid, nil
		}

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, clusterOperatorConditionsValid)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - MachineSet Controller", func() {
	Specify("operator should maintain at least two worker replicas", func() {
		ctx := context.Background()

		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		if !instance.Spec.Features.ReconcileMachineSet {
			Skip("MachineSet Controller is not enabled, skipping this test")
		}

		mss, err := clients.MachineAPI.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(mss.Items).NotTo(BeEmpty())

		// Zero all machinesets, wait for reconcile
		for _, object := range mss.Items {
			err = scale(object.Name, 0)
			Expect(err).NotTo(HaveOccurred())
		}

		for _, object := range mss.Items {
			err = waitForScale(object.Name)
			Expect(err).NotTo(HaveOccurred())
		}

		// Re-count and assert that operator added back replicas
		modifiedMachineSets, err := clients.MachineAPI.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		replicaCount := 0
		for _, machineset := range modifiedMachineSets.Items {
			if machineset.Spec.Replicas != nil {
				replicaCount += int(*machineset.Spec.Replicas)
			}
		}
		Expect(replicaCount).To(BeEquivalentTo(minSupportedReplicas))

		// Scale back to previous state
		for _, ms := range mss.Items {
			err = scale(ms.Name, *ms.Spec.Replicas)
			Expect(err).NotTo(HaveOccurred())
		}

		for _, ms := range mss.Items {
			err = waitForScale(ms.Name)
			Expect(err).NotTo(HaveOccurred())
		}

		// Wait for old machine objects to delete
		err = waitForMachines()
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - Azure Subnet Reconciler", func() {
	var vnetName, location, resourceGroup string
	var subnetsToReconcile map[string]*string
	var testnsg mgmtnetwork.SecurityGroup
	ctx := context.Background()

	const nsg = "e2e-nsg"

	// TODO (robryan) rm this func once default to on https://github.com/Azure/ARO-RP/issues/1735
	enableReconcileSubnet := func() {
		instance, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if !instance.Spec.Features.ReconcileSubnets {
			instance.Spec.Features.ReconcileSubnets = true
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, instance, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	}
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
			Location:                      to.StringPtr(location),
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
		enableReconcileSubnet()
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
