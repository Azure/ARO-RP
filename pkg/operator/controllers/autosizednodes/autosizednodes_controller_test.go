package autosizednodes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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
					operator.AutosizedNodesEnabled: strconv.FormatBool(autoSizeEnabled),
				},
			},
		}
	}

	emptyConfig := mcv1.KubeletConfig{}
	config := makeConfig()

	tests := []struct {
		name       string
		wantErrMsg string
		client     client.Client
		wantConfig *mcv1.KubeletConfig
	}{
		{
			name:       "is not needed",
			client:     testclienthelper.NewAROFakeClientBuilder(aro(false)).Build(),
			wantConfig: &emptyConfig,
			wantErrMsg: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node").Error(),
		},
		{
			name:       "is needed and not present already",
			client:     testclienthelper.NewAROFakeClientBuilder(aro(true)).Build(),
			wantConfig: &config,
		},
		{
			name:       "is needed and present already",
			client:     testclienthelper.NewAROFakeClientBuilder(aro(true), &config).Build(),
			wantConfig: &config,
		},
		{
			name:       "is not needed and is present",
			client:     testclienthelper.NewAROFakeClientBuilder(aro(false), &config).Build(),
			wantConfig: &emptyConfig,
			wantErrMsg: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node").Error(),
		},
		{
			name: "is needed and config got modified",
			client: testclienthelper.NewAROFakeClientBuilder(
				aro(true),
				&mcv1.KubeletConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: configName,
					},
					Spec: mcv1.KubeletConfigSpec{
						AutoSizingReserved: pointerutils.ToPtr(false),
						MachineConfigPoolSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "machineconfiguration.openshift.io/mco-built-in",
									Operator: metav1.LabelSelectorOpExists,
								},
							},
						},
					},
				}).Build(),
			wantConfig: &config,
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
			utilerror.AssertErrorMessage(t, err, test.wantErrMsg)

			if !reflect.DeepEqual(test.wantConfig.Spec, c.Spec) {
				t.Error(cmp.Diff(test.wantConfig.Spec, c.Spec))
			}

			if result != (ctrl.Result{}) {
				t.Error("reconcile returned an unexpected result")
			}
		})
	}
}
