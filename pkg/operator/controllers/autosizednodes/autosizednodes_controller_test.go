package autosizednodes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/google/go-cmp/cmp"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	// This "_" import is counterintuitive but is required to initialize the scheme
	// ARO unfortunately relies on implicit import and its side effect for this
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestAutosizednodesReconciler(t *testing.T) {
	aro := func(autoSizeEnabled bool) *arov1alpha1.Cluster {
		return &arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aro",
				Namespace: "openshift-azure-operator",
			},
			Spec: arov1alpha1.ClusterSpec{
				OperatorFlags: arov1alpha1.OperatorFlags{
					ControllerEnabled: strconv.FormatBool(autoSizeEnabled),
				},
			},
		}
	}

	emptyConfig := mcv1.KubeletConfig{}
	config := makeConfig()

	tests := []struct {
		name       string
		wantGetErr error
		client     client.Client
		wantConfig *mcv1.KubeletConfig
	}{
		{
			name:       "is not needed",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(false)).Build(),
			wantConfig: &emptyConfig,
			wantGetErr: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node"),
		},
		{
			name:       "is needed and not present already",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(true)).Build(),
			wantConfig: &config,
			wantGetErr: nil,
		},
		{
			name:       "is needed and present already",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(true), &config).Build(),
			wantConfig: &config,
		},
		{
			name:       "is not needed and is present",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(false), &config).Build(),
			wantConfig: &emptyConfig,
			wantGetErr: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node"),
		},
		{
			name: "is needed and config got modified",
			client: fake.NewClientBuilder().WithRuntimeObjects(
				aro(true),
				&mcv1.KubeletConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: configName,
					},
					Spec: mcv1.KubeletConfigSpec{
						AutoSizingReserved: util.BoolToPtr(false),
						MachineConfigPoolSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pools.operator.machineconfiguration.openshift.io/worker": "",
							},
						},
					},
				}).Build(),
			wantConfig: &config,
			wantGetErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			r := Reconciler{
				client: test.client,
				log:    logrus.NewEntry(logrus.StandardLogger()),
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "openshift-azure-operator", Name: "aro"}})
			if err != nil {
				t.Error(err)
			}

			key := types.NamespacedName{
				Name: configName,
			}
			var c mcv1.KubeletConfig

			err = r.client.Get(ctx, key, &c)
			if err != nil && err.Error() != test.wantGetErr.Error() || err == nil && test.wantGetErr != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(test.wantConfig.Spec, c.Spec) {
				t.Error(cmp.Diff(test.wantConfig.Spec, c.Spec))
			}

			if result != (ctrl.Result{}) {
				t.Error("reconcile returned an unexpected result")
			}
		})
	}
}
