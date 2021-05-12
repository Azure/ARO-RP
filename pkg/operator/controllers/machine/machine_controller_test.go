package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"testing"

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
	"github.com/Azure/go-autorest/autorest/to"
)

func TestMachineValid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		machine  *machinev1beta1.Machine
		wantErrs []error
	}{
		{
			name: "valid",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
					Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
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
"vmSize": "Standard_D4s_v3"
}`),
						},
					},
				},
			},
		},
		{
			name: "wrong vmSize",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
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
"vmSize": "Standard_D2s_V3"
}`),
						},
					},
				},
			},
			wantErrs: []error{
				errors.New("machine foo-hx8z7-master-0: invalid VM size 'Standard_D2s_V3'"),
			},
		},
		{
			name: "wrong diskSize",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
				},
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(`{
"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
"kind": "AzureMachineProviderSpec",
"osDisk": {
"diskSizeGB": 64
},
"image": {
"publisher": "azureopenshift",
"offer": "aro4"
},
"vmSize": "Standard_D4s_v3"
}`),
						},
					},
				},
			},
			wantErrs: []error{
				errors.New("machine foo-hx8z7-master-0: invalid disk size '64'"),
			},
		},
		{
			name: "wrong image publisher",
			machine: &machinev1beta1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-hx8z7-master-0",
					Namespace: machineSetsNamespace,
					Labels:    map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "MachineSet",
						},
					},
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
"publisher": "xyzcorp",
"offer": "bananas"
},
"vmSize": "Standard_D4s_v3"
}`),
						},
					},
				},
			},
			wantErrs: []error{
				errors.New("machine foo-hx8z7-master-0: invalid image '{xyzcorp bananas   }'"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MachineReconciler{}

			isMaster, err := isMasterRole(tt.machine)
			if err != nil {
				t.Fatal(err)
			}

			errs := r.machineValid(ctx, tt.machine, isMaster)

			if !reflect.DeepEqual(errs, tt.wantErrs) {
				t.Errorf("MachineReconciler.machineValid() = %v, want %v", errs, tt.wantErrs)
			}
		})
	}
}

func TestMachineReconciler(t *testing.T) {
	// pass in fake machines to be passed into Reconciler
	newFakeMao := func() *maofake.Clientset {
		master0 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-master-0",
				Namespace: machineSetsNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "MachineSet",
					},
				},
				Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
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
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "MachineSet",
					},
				},
				Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
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
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "MachineSet",
					},
				},
				Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "master"},
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
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "MachineSet",
					},
				},
				Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
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

		worker1 := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-hx8z7-worker-1",
				Namespace: machineSetsNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "MachineSet",
					},
				},
				Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
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
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "MachineSet",
					},
				},
				Labels: map[string]string{"machine.openshift.io/cluster-api-machine-role": "worker"},
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

	// define fake cluster resource
	newFakeAro := func(a *arov1alpha1.Cluster) *arofake.Clientset {
		return arofake.NewSimpleClientset(a)
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
		wantErr        bool
	}{
		{
			name:   "valid",
			maocli: newFakeMao(),
			arocli: newFakeAro(&baseCluster),
			wantConditions: []status.Condition{{
				Type:    arov1alpha1.MachineValid,
				Status:  corev1.ConditionTrue,
				Message: "All machines valid",
				Reason:  "CheckDone",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := &MachineReconciler{
				maocli:                 tt.maocli,
				log:                    logrus.NewEntry(logrus.StandardLogger()),
				arocli:                 tt.arocli,
				isLocalDevelopmentMode: false,
				role:                   "master",
			}

			_, err := r.Reconcile(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Machine.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			cluster, err := r.arocli.AroV1alpha1().Clusters().Get(context.Background(), arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				t.Fatal("Error found")
			}

			// Fix this to compare without timestamp
			if !reflect.DeepEqual(cluster.Status.Conditions, tt.wantConditions) {
				t.Fatalf("Unexpected condition found\n want: %v\n got: %v", tt.wantConditions, cluster.Status.Conditions)
			}

		})
	}
}
