package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	operatorv1 "github.com/openshift/api/operator/v1"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestMachineReconciler(t *testing.T) {
	newFakeMao := func(diskSize string, imagePublisher string, vmSize string) *maofake.Clientset {
		master0 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-master-0",
				Namespace: machineSetsNamespace,
				Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &runtime.RawExtension{
						Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 512
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D8s_v3"
}`)},
				},
			},
		}

		master1 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo1-hx8z7-master-1",
				Namespace: machineSetsNamespace,
				Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &runtime.RawExtension{
						Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 512
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D8s_v3"
}`)},
				},
			},
		}

		master2 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-master-2",
				Namespace: machineSetsNamespace,
				Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &runtime.RawExtension{
						Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 512
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D8s_v3"
}`)},
				},
			},
		}

		worker0 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-worker-0",
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
"offer": "aro4"
},
"vmSize": "%v"
}`, diskSize, imagePublisher, vmSize))},
				},
			},
		}

		worker1 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-worker-1",
				Namespace: machineSetsNamespace,
				Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &runtime.RawExtension{
						Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 128
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D4s_v3"
}`)},
				},
			},
		}

		worker2 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-worker-2",
				Namespace: machineSetsNamespace,
				Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: machinev1beta1.ProviderSpec{
					Value: &runtime.RawExtension{
						Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 128
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D4s_v3"
}`)},
				},
			},
		}

		workerMachineSet := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "workermachineset",
				Namespace: machineSetsNamespace,
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(3),
			},
		}
		return maofake.NewSimpleClientset(worker0, worker1, worker2, master0, master1, master2, workerMachineSet)
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     arov1alpha1.ClusterStatus{Conditions: []operatorv1.OperatorCondition{}},
	}

	tests := []struct {
		name           string
		request        ctrl.Request
		maocli         *maofake.Clientset
		arocli         *arofake.Clientset
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name:   "valid",
			maocli: newFakeMao("512", "azureopenshift", "Standard_D4s_v3"),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid.String(),
				Status:  operatorv1.ConditionTrue,
				Message: "All machines valid",
				Reason:  "CheckDone",
			}},
		},
		{
			name:   "wrong vm size",
			maocli: newFakeMao("512", "azureopenshift", "Standard_D4s_v9"),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid.String(),
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-0: invalid VM size 'Standard_D4s_v9'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "wrong disk size",
			maocli: newFakeMao("64", "azureopenshift", "Standard_D4s_v3"),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid.String(),
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-0: invalid disk size '64'",
				Reason:  "CheckFailed",
			}},
		},
		{
			name:   "wrong image publisher",
			maocli: newFakeMao("512", "bananas", "Standard_D4s_v3"),
			wantConditions: []operatorv1.OperatorCondition{{
				Type:    arov1alpha1.MachineValid.String(),
				Status:  operatorv1.ConditionFalse,
				Message: "machine foo-hx8z7-worker-0: invalid image '{bananas aro4   }'",
				Reason:  "CheckFailed",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MachineReconciler{
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
