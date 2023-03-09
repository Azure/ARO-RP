package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestMachineReconciler(t *testing.T) {
	// Fake cluster with AZs
	newFakeMao1 := func(diskSize, imagePublisher, vmSize, masterVmSize string) []client.Object {
		master0 := getValidMachine("foo-hx8z7-master-0", "", "", "", "", true)
		master1 := getValidMachine("foo-hx8z7-master-1", "", "", "", "", true)
		master2 := getValidMachine("foo-hx8z7-master-2", "", "", masterVmSize, "", true)
		worker0 := getValidMachine("foo-hx8z7-worker-0", diskSize, "", "", "", false)
		worker1 := getValidMachine("foo-hx8z7-worker-1", "", imagePublisher, "", "", false)
		worker2 := getValidMachine("foo-hx8z7-worker-2", "", "", vmSize, "", false)
		workerMachineSet0 := workerMachineSet("foo-hx8z7-machineset-0")
		workerMachineSet1 := workerMachineSet("foo-hx8z7-machineset-1")
		workerMachineSet2 := workerMachineSet("foo-hx8z7-machineset-2")

		return []client.Object{worker0, worker1, worker2, master0, master1, master2, workerMachineSet0, workerMachineSet1, workerMachineSet2}
	}

	// Fake cluster missing a master
	newFakeMao2 := func() []client.Object {
		master0 := getValidMachine("foo-hx8z7-master-0", "", "", "", "", true)
		master2 := getValidMachine("foo-hx8z7-master-2", "", "", "", "", true)
		worker0 := getValidMachine("foo-hx8z7-worker-0", "", "", "", "", false)
		worker1 := getValidMachine("foo-hx8z7-worker-1", "", "", "", "", false)
		worker2 := getValidMachine("foo-hx8z7-worker-2", "", "", "", "", false)
		workerMachineSet0 := workerMachineSet("foo-hx8z7-machineset-0")
		workerMachineSet1 := workerMachineSet("foo-hx8z7-machineset-1")
		workerMachineSet2 := workerMachineSet("foo-hx8z7-machineset-2")

		return []client.Object{worker0, worker1, worker2, master0, master2, workerMachineSet0, workerMachineSet1, workerMachineSet2}
	}

	// Fake cluster missing a worker
	newFakeMao3 := func() []client.Object {
		master0 := getValidMachine("foo-hx8z7-master-0", "", "", "", "", true)
		master1 := getValidMachine("foo-hx8z7-master-1", "", "", "", "", true)
		master2 := getValidMachine("foo-hx8z7-master-2", "", "", "", "", true)
		worker0 := getValidMachine("foo-hx8z7-worker-0", "", "", "", "", false)
		worker1 := getValidMachine("foo-hx8z7-worker-1", "", "", "", "", false)
		workerMachineSet0 := workerMachineSet("foo-hx8z7-machineset-0")
		workerMachineSet1 := workerMachineSet("foo-hx8z7-machineset-1")
		workerMachineSet2 := workerMachineSet("foo-hx8z7-machineset-2")

		return []client.Object{worker0, worker1, master0, master1, master2, workerMachineSet0, workerMachineSet1, workerMachineSet2}
	}

	tests := []struct {
		name           string
		request        ctrl.Request
		objects        []client.Object
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name:    "valid",
			objects: newFakeMao1("", "", "", ""),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionTrue,
				Message: "All machines valid",
				Reason:  "CheckDone",
			}},
		},
		{
			name:    "wrong vm size",
			objects: newFakeMao1("", "", "Standard_D4s_v9", ""),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-2: invalid VM size 'Standard_D4s_v9'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:    "wrong disk size",
			objects: newFakeMao1("64", "", "", ""),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-0: invalid disk size '64'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:    "wrong image publisher",
			objects: newFakeMao1("", "bananas", "", ""),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-1: invalid image '{bananas aro4    }'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:    "wrong vm size on master",
			objects: newFakeMao1("", "", "", "Standard_D4s_v9"),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-master-2: invalid VM size 'Standard_D4s_v9'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:    "invalid master machine count",
			objects: newFakeMao2(),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionFalse,
				Message: "invalid number of master machines 2, expected 3",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:    "invalid worker machine count",
			objects: newFakeMao3(),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid,
				Status:  operatorv1.ConditionFalse,
				Message: "invalid number of worker machines 2, expected 3",
				Reason:  "CheckFailed",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			baseCluster := arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
					},
				},
			}

			clientFake := fake.NewClientBuilder().WithObjects(&baseCluster).WithObjects(tt.objects...).Build()

			r := &Reconciler{
				log:                    logrus.NewEntry(logrus.StandardLogger()),
				isLocalDevelopmentMode: false,
				role:                   "master",
				client:                 clientFake,
			}

			_, err := r.Reconcile(ctx, tt.request)
			if err != nil {
				t.Fatal(err)
			}

			cluster := &arov1alpha1.Cluster{}
			err = r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
			if err != nil {
				t.Fatal(err)
			}

			if cluster.Status.Conditions[0].Type != tt.wantConditions[0].Type {
				t.Error(cluster.Status.Conditions[0].Type)
			}

			if cluster.Status.Conditions[0].Status != tt.wantConditions[0].Status {
				t.Error(cluster.Status.Conditions[0].Status)
			}

			if strings.TrimSpace(cluster.Status.Conditions[0].Message) != tt.wantConditions[0].Message {
				t.Error(cluster.Status.Conditions[0].Message)
			}

			if cluster.Status.Conditions[0].Reason != tt.wantConditions[0].Reason {
				t.Error(cluster.Status.Conditions[0].Reason)
			}
		})
	}
}

func getValidMachine(name, diskSize, imagePublisher, vmSize, offer string, isMaster bool) *machinev1beta1.Machine {
	if diskSize == "" {
		if isMaster {
			diskSize = "512"
		} else {
			diskSize = "128"
		}
	}
	if imagePublisher == "" {
		imagePublisher = "azureopenshift"
	}
	if vmSize == "" {
		if isMaster {
			vmSize = "Standard_D8s_v3"
		} else {
			vmSize = "Standard_D4s_v3"
		}
	}

	if offer == "" {
		offer = "aro4"
	}
	labels := map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"}
	if isMaster {
		labels = map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"}
	}

	// To check that we support both API versions of AzureMachineProviderSpec
	apiVersion := "azureproviderconfig.openshift.io/v1beta1"
	if isMaster {
		apiVersion = "machine.openshift.io/v1beta1"
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineSetsNamespace,
			Labels:    labels,
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &kruntime.RawExtension{
					Raw: []byte(fmt.Sprintf(`{
"apiVersion": "%v",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": %v
},
"image": {
"publisher": "%v",
"offer": "%v"
},
"vmSize": "%v"
}`, apiVersion, diskSize, imagePublisher, offer, vmSize))},
			},
		},
	}
}

func workerMachineSet(name string) *machinev1beta1.MachineSet {
	return &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineSetsNamespace,
		},
		Spec: machinev1beta1.MachineSetSpec{
			Replicas: to.Int32Ptr(1),
		},
	}
}
