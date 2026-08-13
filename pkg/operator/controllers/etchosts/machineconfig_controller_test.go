package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestReconcileEtcHostsMachineConfig(t *testing.T) {
	type test struct {
		name           string
		objects        []client.Object
		createdObjects map[string]int
		updatedObjects map[string]int
		expectedLog    []testlog.ExpectedLogEntry
		requestName    string
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
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("controller is disabled"),
				},
			},
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed=false, deletions not handled in this controller",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalse,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("etchosts are not managed by this controller"),
				},
			},
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, regex not match",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
			},
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, regex match",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			updatedObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("allowing reconciliation of EtcHostsMachineConfig because reconciliation forced"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile object openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.HavePrefix("Update MachineConfig.machineconfiguration.openshift.io/99-master-aro-etc-hosts-gateway-domains: "),
				},
			},
			requestName: "99-master-aro-etc-hosts-gateway-domains",
		},
		{
			name: "etchosts controller enabled, managed false, cluster not updating, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseForceReconcileFalse, clusterVersionNotUpdating, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("etchosts are not managed by this controller"),
				},
			},
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, cluster not updating, regex not match, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
			},
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, cluster updating, regex not match, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/cluster"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
			},
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, cluster not updating, regex match, reconcile not forced - no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"),
				},
			},
			requestName: "99-master-aro-etc-hosts-gateway-domains",
		},
		{
			name: "etchosts controller enabled, managed true, cluster updating, regex match, reconcile not forced - ensure machine config",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledForceReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			updatedObjects: map[string]int{
				"MachineConfig//99-master-aro-etc-hosts-gateway-domains": 1,
			},
			expectedLog: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile MachineConfig openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("running"),
				},
				{
					"level": gomega.Equal(logrus.DebugLevel),
					"msg":   gomega.Equal("reconcile object openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.HavePrefix("Update MachineConfig.machineconfiguration.openshift.io/99-master-aro-etc-hosts-gateway-domains: "),
				},
			},
			requestName: "99-master-aro-etc-hosts-gateway-domains",
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

			r := &EtcHostsMachineConfigReconciler{
				AROController: base.AROController{
					Log:    logger,
					Client: ch,
					Name:   ControllerName,
				},
				ch: clienthelper.NewWithClient(logger, ch),
			}

			request := ctrl.Request{}
			request.Name = tt.requestName

			_, err := r.Reconcile(t.Context(), request)
			if err != nil {
				logger.Log(logrus.ErrorLevel, err)
			}

			err = testlog.AssertLoggingOutput(hook, tt.expectedLog)
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

			errs, err = testclienthelper.CompareTally(tt.updatedObjects, updatedObjects)
			if err != nil {
				t.Error(err, "on updated objects")
				for _, l := range errs {
					t.Error(l)
				}
			}

			if len(deletedObjects) != 0 {
				t.Error("should not have deleted objects", deletedObjects)
			}
		})
	}
}
