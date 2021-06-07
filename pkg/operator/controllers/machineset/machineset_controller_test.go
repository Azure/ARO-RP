package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestMachineReconciler(t *testing.T) {
	newFakeMao := func(replicas int32) *maofake.Clientset {
		workerMachineSet := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: machineSetsNamespace,
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(replicas), // Modify replicas accordingly
			},
		}
		return maofake.NewSimpleClientset(workerMachineSet)
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
	}

	tests := []struct {
		name    string
		request ctrl.Request
		maocli  *maofake.Clientset
		wantErr string
	}{
		{
			name:    "one worker replica",
			maocli:  newFakeMao(1),
			wantErr: "Found less than 3 worker replicas. The MachineSet controller will attempt scaling.",
		},
		{
			name:    "two worker replicas",
			maocli:  newFakeMao(2),
			wantErr: "Found less than 3 worker replicas. The MachineSet controller will attempt scaling.",
		},
		{
			name:    "three worker replicas",
			maocli:  newFakeMao(3),
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MachineSetReconciler{
				maocli: tt.maocli,
				log:    logrus.NewEntry(logrus.StandardLogger()),
				arocli: arofake.NewSimpleClientset(&baseCluster),
			}

			_, err := r.Reconcile(context.Background(), tt.request)
			if err != nil && err.Error() != tt.wantErr {
				t.Fatalf("Unexpected error:\nwant: %s\ngot: %s", tt.wantErr, err.Error())
			}
		})
	}
}
