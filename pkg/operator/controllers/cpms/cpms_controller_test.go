package cpms

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	machinev1 "github.com/openshift/api/machine/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestReconcile(t *testing.T) {
	for _, tt := range []struct {
		name       string
		enabled    bool
		cpms       *machinev1.ControlPlaneMachineSet
		wantDelete bool
	}{
		{
			name:    "cpms enabled, does nothing",
			enabled: true,
			cpms: &machinev1.ControlPlaneMachineSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SingletonCPMSName,
					Namespace: SingletonCPMSNamespace,
				},
				Spec: machinev1.ControlPlaneMachineSetSpec{
					State: machinev1.ControlPlaneMachineSetStateActive,
				},
			},
		},
		{
			name:    "no CPMS, does nothing",
			enabled: false,
		},
		{
			name:    "CPMS inactive, does nothing",
			enabled: false,
			cpms: &machinev1.ControlPlaneMachineSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SingletonCPMSName,
					Namespace: SingletonCPMSNamespace,
				},
				Spec: machinev1.ControlPlaneMachineSetSpec{
					State: machinev1.ControlPlaneMachineSetStateInactive,
				},
			},
		},
		{
			name:    "CPMS active, deletes",
			enabled: false,
			cpms: &machinev1.ControlPlaneMachineSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SingletonCPMSName,
					Namespace: SingletonCPMSNamespace,
				},
				Spec: machinev1.ControlPlaneMachineSetSpec{
					State: machinev1.ControlPlaneMachineSetStateActive,
				},
			},
			wantDelete: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.StandardLogger())

			operatorFlag := operator.FlagFalse
			if tt.enabled {
				operatorFlag = operator.FlagTrue
			}

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.CPMSEnabled: operatorFlag,
					},
				},
			}

			clientBuilder := testclienthelper.NewAROFakeClientBuilder(instance)

			if tt.cpms != nil {
				clientBuilder.WithObjects(tt.cpms)
			}

			client := clientBuilder.Build()

			r := NewReconciler(log, client)

			ctx := context.Background()
			_, err := r.Reconcile(ctx, ctrl.Request{})
			utilerror.AssertErrorMessage(t, err, "")

			cpms := &machinev1.ControlPlaneMachineSet{}
			err = client.Get(
				ctx,
				types.NamespacedName{Name: SingletonCPMSName, Namespace: SingletonCPMSNamespace},
				cpms,
			)

			if tt.wantDelete {
				if err == nil || !kerrors.IsNotFound(err) {
					t.Errorf("CPMS still present on cluster")
				}
			} else {
				if tt.cpms != nil && kerrors.IsNotFound(err) {
					t.Errorf("Did not want delete on CPMS but was deleted")
				}
			}
		})
	}
}
