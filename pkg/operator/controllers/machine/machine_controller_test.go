package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestMachineReconciler(t *testing.T) {
	// Fake cluster with AZs
	newFakeMao1 := func(diskSize, imagePublisher, vmSize, masterVmSize string) *maofake.Clientset {
		master0 := masterMachine("foo-hx8z7-master-0", "", "", "", "")
		master1 := masterMachine("foo-hx8z7-master-1", "", "", "", "")
		master2 := masterMachine("foo-hx8z7-master-2", "", "", masterVmSize, "")
		worker0 := workerMachine("foo-hx8z7-worker-0", diskSize, "", "", "")
		worker1 := workerMachine("foo-hx8z7-worker-1", "", imagePublisher, "", "")
		worker2 := workerMachine("foo-hx8z7-worker-2", "", "", vmSize, "")
		workerMachineSet0 := workerMachineSet("foo-hx8z7-machineset-0")
		workerMachineSet1 := workerMachineSet("foo-hx8z7-machineset-1")
		workerMachineSet2 := workerMachineSet("foo-hx8z7-machineset-2")

		return maofake.NewSimpleClientset(worker0, worker1, worker2, master0, master1, master2, workerMachineSet0, workerMachineSet1, workerMachineSet2)
	}

	// Fake cluster missing a master
	newFakeMao2 := func() *maofake.Clientset {
		master0 := masterMachine("foo-hx8z7-master-0", "", "", "", "")
		master2 := masterMachine("foo-hx8z7-master-2", "", "", "", "")
		worker0 := workerMachine("foo-hx8z7-worker-0", "", "", "", "")
		worker1 := workerMachine("foo-hx8z7-worker-1", "", "", "", "")
		worker2 := workerMachine("foo-hx8z7-worker-2", "", "", "", "")
		workerMachineSet0 := workerMachineSet("foo-hx8z7-machineset-0")
		workerMachineSet1 := workerMachineSet("foo-hx8z7-machineset-1")
		workerMachineSet2 := workerMachineSet("foo-hx8z7-machineset-2")

		return maofake.NewSimpleClientset(worker0, worker1, worker2, master0, master2, workerMachineSet0, workerMachineSet1, workerMachineSet2)
	}

	// Fake cluster missing a worker
	newFakeMao3 := func() *maofake.Clientset {
		master0 := masterMachine("foo-hx8z7-master-0", "", "", "", "")
		master1 := masterMachine("foo-hx8z7-master-1", "", "", "", "")
		master2 := masterMachine("foo-hx8z7-master-2", "", "", "", "")
		worker0 := workerMachine("foo-hx8z7-worker-0", "", "", "", "")
		worker1 := workerMachine("foo-hx8z7-worker-1", "", "", "", "")
		workerMachineSet0 := workerMachineSet("foo-hx8z7-machineset-0")
		workerMachineSet1 := workerMachineSet("foo-hx8z7-machineset-1")
		workerMachineSet2 := workerMachineSet("foo-hx8z7-machineset-2")

		return maofake.NewSimpleClientset(worker0, worker1, master0, master1, master2, workerMachineSet0, workerMachineSet1, workerMachineSet2)
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     arov1alpha1.ClusterStatus{Conditions: []status.Condition{}},
	}

	tests := []struct {
		name           string
		request        ctrl.Request
		maocli         *maofake.Clientset
		arocli         *arofake.Clientset
		wantConditions []status.Condition
	}{
		{
			name:   "valid",
			maocli: newFakeMao1("", "", "", ""),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionTrue,
				Message: "All machines valid",
				Reason:  "CheckDone",
			}},
		},
		{
			name:   "wrong vm size",
			maocli: newFakeMao1("", "", "Standard_D4s_v9", ""),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-2: invalid VM size 'Standard_D4s_v9'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "wrong disk size",
			maocli: newFakeMao1("64", "", "", ""),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-0: invalid disk size '64'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "wrong image publisher",
			maocli: newFakeMao1("", "bananas", "", ""),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-1: invalid image '{bananas aro4   }'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "wrong vm size on master",
			maocli: newFakeMao1("", "", "", "Standard_D4s_v9"),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionFalse,
				Message: "machine foo-hx8z7-master-2: invalid VM size 'Standard_D4s_v9'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "invalid master machine count",
			maocli: newFakeMao2(),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionFalse,
				Message: "invalid number of master machines 2, expected 3",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "invalid worker machine count",
			maocli: newFakeMao3(),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionFalse,
				Message: "invalid number of worker machines 2, expected 3",
				Reason:  "CheckFailed",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				maocli:                 tt.maocli,
				log:                    logrus.NewEntry(logrus.StandardLogger()),
				arocli:                 arofake.NewSimpleClientset(&baseCluster),
				isLocalDevelopmentMode: false,
				role:                   "master",
			}

			_, err := r.Reconcile(context.Background(), tt.request)
			if err != nil {
				t.Fatal(err)
			}

			cluster, err := r.arocli.AroV1alpha1().Clusters().Get(context.Background(), arov1alpha1.SingletonClusterName, metav1.GetOptions{})
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

func masterMachine(name, diskSize, imagePublisher, vmSize, offer string) *machinev1beta1.Machine {
	if diskSize == "" {
		diskSize = "512"
	}
	if imagePublisher == "" {
		imagePublisher = "azureopenshift"
	}
	if vmSize == "" {
		vmSize = "Standard_D8s_v3"
	}
	if offer == "" {
		offer = "aro4"
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineSetsNamespace,
			Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": %v
},
"image": {
"publisher": "%v",
"offer": "%v"
},
"vmSize": "%v"
}`, diskSize, imagePublisher, offer, vmSize))},
			},
		},
	}
}

func workerMachine(name, diskSize, imagePublisher, vmSize, offer string) *machinev1beta1.Machine {
	if diskSize == "" {
		diskSize = "128"
	}
	if imagePublisher == "" {
		imagePublisher = "azureopenshift"
	}
	if vmSize == "" {
		vmSize = "Standard_D4s_v3"
	}
	if offer == "" {
		offer = "aro4"
	}

	return &machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: machineSetsNamespace,
			Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
		},
		Spec: machinev1beta1.MachineSpec{
			ProviderSpec: machinev1beta1.ProviderSpec{
				Value: &runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": %v
},
"image": {
"publisher": "%v",
"offer": "%v"
},
"vmSize": "%v"
}`, diskSize, imagePublisher, offer, vmSize))},
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
