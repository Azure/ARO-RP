package autosizednodes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
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
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
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

	configWithOtherAnnotations := makeConfig()
	configWithOtherAnnotations.Annotations = map[string]string{
		"some-other-annotation": "value",
	}

	configWithOtherAnnotationsAndMCO := makeConfig()
	configWithOtherAnnotationsAndMCO.Annotations = map[string]string{
		"some-other-annotation":                            "value",
		"machineconfiguration.openshift.io/mc-name-suffix": "",
	}

	configWithAnnotation := makeConfig()
	configWithAnnotation.Annotations = map[string]string{
		"machineconfiguration.openshift.io/mc-name-suffix": "",
	}

	configWithHighAnnotation := makeConfig()
	configWithHighAnnotation.Annotations = map[string]string{
		"machineconfiguration.openshift.io/mc-name-suffix": "3",
	}

	tests := []struct {
		name       string
		wantErrMsg string
		client     client.Client
		wantConfig *mcv1.KubeletConfig
	}{
		{
			name:       "is not needed",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(false)).Build(),
			wantConfig: &emptyConfig,
			wantErrMsg: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node").Error(),
		},
		{
			name:   "is needed and not present already",
			client: fake.NewClientBuilder().WithRuntimeObjects(aro(true)).Build(),
			// We don't create the KubeletConfig with the annotation because it gets updated by MCO, and we
			wantConfig: &config,
		},
		{
			name:       "is needed and present already",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(true), &configWithAnnotation).Build(),
			wantConfig: &configWithAnnotation,
		},
		{
			name:       "is not needed and is present",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(false), &config).Build(),
			wantConfig: &emptyConfig,
			wantErrMsg: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node").Error(),
		},
		{
			name: "is needed and config got modified",
			client: fake.NewClientBuilder().WithRuntimeObjects(
				aro(true),
				&mcv1.KubeletConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: configName,
						Annotations: map[string]string{
							"machineconfiguration.openshift.io/mc-name-suffix": "",
						},
					},
					Spec: mcv1.KubeletConfigSpec{
						AutoSizingReserved: to.BoolPtr(false),
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
			wantConfig: &configWithAnnotation,
		},
		{
			name:       "is needed and present already, only kubelet config, gets MCO annotation set",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(true), &config).Build(),
			wantConfig: &configWithAnnotation,
		},
		{
			name:       "is needed and present already, non-nil annotations, only kubelet config, gets MCO annotation set",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(true), &configWithOtherAnnotations).Build(),
			wantConfig: &configWithOtherAnnotationsAndMCO,
		},
		{
			name:       "is needed and present already, only kubelet config, has MCO annotation already, annotation remains",
			client:     fake.NewClientBuilder().WithRuntimeObjects(aro(true), &configWithHighAnnotation).Build(),
			wantConfig: &configWithHighAnnotation,
		},
		{
			name: "is needed and present already, several kubelet configs without MCO annotations, gets MCO annotation set",
			client: fake.NewClientBuilder().WithRuntimeObjects(
				aro(true),
				&config,
				&mcv1.KubeletConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-other-config",
					},
					Spec: mcv1.KubeletConfigSpec{},
				}).Build(),
			wantConfig: &configWithAnnotation,
		},
		{
			name: "is needed and present already, annotations present, several kubelet configs without MCO annotations, gets MCO annotation set",
			client: fake.NewClientBuilder().WithRuntimeObjects(
				aro(true),
				&configWithOtherAnnotations,
				&mcv1.KubeletConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-other-config",
					},
					Spec: mcv1.KubeletConfigSpec{},
				}).Build(),
			wantConfig: &configWithOtherAnnotationsAndMCO,
		},
		{
			name: "is needed and present already, several kubelet configs, annotations exist, MCO annotation not set",
			client: fake.NewClientBuilder().WithRuntimeObjects(
				aro(true),
				&config,
				&mcv1.KubeletConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-other-config",
						Annotations: map[string]string{
							"machineconfiguration.openshift.io/mc-name-suffix": "",
						},
					},
					Spec: mcv1.KubeletConfigSpec{},
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
			if !reflect.DeepEqual(test.wantConfig.Annotations, c.Annotations) {
				t.Error(cmp.Diff(test.wantConfig.Annotations, c.Annotations))
			}

			if result != (ctrl.Result{}) {
				t.Error("reconcile returned an unexpected result")
			}
		})
	}
}
