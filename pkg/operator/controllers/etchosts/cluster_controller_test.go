package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	mcv1 "github.com/openshift/api/machineconfiguration/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var (
	clusterEtcHostsControllerDisabled = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagFalse,
				operator.ForceReconciliation: operator.FlagTrue,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabled = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagTrue,
				operator.ForceReconciliation: operator.FlagTrue,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabledManagedFalse = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagFalse,
				operator.ForceReconciliation: operator.FlagTrue,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabledForceReconcileFalse = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagTrue,
				operator.ForceReconciliation: operator.FlagFalse,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabledManagedFalseForceReconcileFalse = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagFalse,
				operator.ForceReconciliation: operator.FlagFalse,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	machinePoolMaster = &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{Name: "master"},
		Status:     mcv1.MachineConfigPoolStatus{},
		Spec:       mcv1.MachineConfigPoolSpec{},
	}
	machinePoolWorker = &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{Name: "worker"},
		Status:     mcv1.MachineConfigPoolStatus{},
		Spec:       mcv1.MachineConfigPoolSpec{},
	}
	clusterVersionNotUpdating = &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: "4.10.11",
				},
			},
		},
	}
	clusterVersionUpdating = &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.PartialUpdate,
					Version: "4.19.0",
				},
				{
					State:   configv1.CompletedUpdate,
					Version: "4.10.11",
				},
			},
		},
	}
)

func TestReconcileEtcHostsCluster(t *testing.T) {
	type test struct {
		name           string
		objects        []client.Object
		createdObjects map[string]int
		deletedObjects map[string]int
		expectedLog    []testlog.ExpectedLogEntry
	}

	for _, tt := range []*test{
		{
			name: "etchosts controller disabled",
			objects: []client.Object{
				clusterEtcHostsControllerDisabled,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("controller is disabled"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed false",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalse, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			deletedObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
				"MachineConfig//99-worker-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("etchosts managed is false, removing machine configs"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("removing machine config 99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("removing machine config 99-worker-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("etchosts managed is false, machine configs removed"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, only master mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata,
			},
			createdObjects: map[string]int{
				"MachineConfig//99-worker-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-worker-aro-etc-hosts-gateway-domains"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, only worker mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsWorkerMCMetadata,
			},
			createdObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-master-aro-etc-hosts-gateway-domains"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, no mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker,
			},
			createdObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
				"MachineConfig//99-worker-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-worker-aro-etc-hosts-gateway-domains"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed false, force reconcile false, cluster not updating, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reboot-causing reconciliation not allowed right now"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed false, force reconcile false, cluster updating, mc removed",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			deletedObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
				"MachineConfig//99-worker-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("etchosts managed is false, removing machine configs"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("removing machine config 99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("removing machine config 99-worker-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("etchosts managed is false, machine configs removed"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster not updating, mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reboot-causing reconciliation not allowed right now"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster updating, mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster not updating, only master mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reboot-causing reconciliation not allowed right now"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster updating, only master mc exist, ensure worker mc",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata,
			},
			createdObjects: map[string]int{
				"MachineConfig//99-worker-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-worker-aro-etc-hosts-gateway-domains"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster not updating, only worker mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reboot-causing reconciliation not allowed right now"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster updating, only worker mc exist, ensure master mc",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsWorkerMCMetadata,
			},
			createdObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-master-aro-etc-hosts-gateway-domains"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster not updating, no mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reboot-causing reconciliation not allowed right now"),
				},
			},
		},
		{
			name: "etchosts controller enabled, managed true, force reconcile false, cluster updating, no mc exist, ensure master and worker mc",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker,
			},
			createdObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
				"MachineConfig//99-worker-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Create MachineConfig.machineconfiguration.openshift.io/99-worker-aro-etc-hosts-gateway-domains"),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hook, logger := testlog.LogForTesting(t)
			logger.Logger.SetLevel(logrus.TraceLevel)

			createdObjects := map[string]int{}
			updatedObjects := map[string]int{}
			deletedObjects := map[string]int{}

			clientBuilder := testclienthelper.NewAROFakeClientBuilder(tt.objects...)
			ch := testclienthelper.NewHookingClient(clientBuilder.Build())
			ch.WithPostCreateHook(testclienthelper.TallyCountsAndKey(createdObjects))
			ch.WithPostDeleteHook(testclienthelper.TallyCountsAndKey(deletedObjects))
			ch.WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updatedObjects))

			r := &EtcHostsClusterReconciler{
				AROController: base.AROController{
					Log:    logger,
					Client: ch,
					Name:   ControllerName,
				},
				ch: clienthelper.NewWithClient(logger, ch),
			}

			request := ctrl.Request{}
			request.Name = "cluster"

			_, err := r.Reconcile(t.Context(), request)
			if err != nil {
				logger.Log(logrus.ErrorLevel, err)
			}

			logs := []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
			}
			logs = append(logs, tt.expectedLog...)

			err = testlog.AssertLoggingOutput(hook, logs)
			if err != nil {
				t.Error(err)
			}

			errs, err := testclienthelper.CompareTally(tt.createdObjects, createdObjects)
			if err != nil {
				t.Error(err, "on created objects")
				for _, l := range errs {
					t.Error(l)
				}
			}

			errs, err = testclienthelper.CompareTally(tt.deletedObjects, deletedObjects)
			if err != nil {
				t.Error(err, "on deleted objects")
				for _, l := range errs {
					t.Error(l)
				}
			}

			if len(updatedObjects) != 0 {
				t.Error("no objects should be updated", updatedObjects)
			}
		})
	}
}
