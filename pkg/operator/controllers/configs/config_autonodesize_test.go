package configs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	// This import is counterintuitive but is required to initialize the scheme
	// ARO unfortunately relies on implicit import and its side effect for this
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestIsApplicable(t *testing.T) {
	aroMeta := metav1.ObjectMeta{
		Name:      "aro",
		Namespace: "openshift-azure-operator",
	}

	kubeletConfig := mcv1.KubeletConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeletConfiguration",
			APIVersion: "kubelet.config.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
	}

	tests := []struct {
		name     string
		wantBool bool
		aro      arov1alpha1.Cluster
		client   client.Client
	}{
		{
			name: "is applicable",
			aro: arov1alpha1.Cluster{
				ObjectMeta: aroMeta,
				Spec: arov1alpha1.ClusterSpec{
					Features: arov1alpha1.FeaturesSpec{
						ReconcileAutoSizedNodes: true,
					},
				},
			},
			client:   fake.NewClientBuilder().WithRuntimeObjects().Build(),
			wantBool: true,
		},
		{
			name:     "is already applied",
			wantBool: false,
			aro: arov1alpha1.Cluster{
				ObjectMeta: aroMeta,
				Spec: arov1alpha1.ClusterSpec{
					Features: arov1alpha1.FeaturesSpec{
						ReconcileAutoSizedNodes: true,
					},
				},
			},
			client: fake.NewClientBuilder().WithRuntimeObjects(&kubeletConfig).Build(),
		},
		{
			name:     "is not applicable",
			wantBool: false,
			aro: arov1alpha1.Cluster{
				ObjectMeta: aroMeta,
				Spec:       arov1alpha1.ClusterSpec{},
			},
			client: fake.NewClientBuilder().WithRuntimeObjects().Build(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := NewAutoNodeSizeConfig()
			ctx := context.Background()

			r := Reconciler{
				Client: test.client,
			}

			result := config.IsApplicable(test.aro, &r, ctx)
			if result != test.wantBool {
				t.Error("isApplicable did not returned expected result")
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	tests := []struct {
		name       string
		wantConfig *mcv1.KubeletConfig
		client     client.Client
	}{
		{
			name:       "Ensured successfully",
			wantConfig: makeConfig(),
			client:     fake.NewClientBuilder().WithRuntimeObjects().Build(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := NewAutoNodeSizeConfig()
			ctx := context.Background()

			r := Reconciler{
				Client: test.client,
			}

			err := config.Ensure(&r, ctx)
			if err != nil {
				t.Error("Ensure could not apply the change")
			}

			key := types.NamespacedName{
				Name: configName,
			}
			var c mcv1.KubeletConfig

			err = r.Get(ctx, key, &c)
			if err != nil {
				t.Error("Could not verify config presence")
			}

			if !reflect.DeepEqual(test.wantConfig.Spec, c.Spec) {
				t.Error("The applied config does not match")
			}

		})
	}
}

func TestRemove(t *testing.T) {
	config := makeConfig()

	tests := []struct {
		name    string
		wantErr error
		client  client.Client
	}{
		{
			name:    "Removed successfully",
			wantErr: kerrors.NewNotFound(mcv1.Resource("kubeletconfigs"), "dynamic-node"),
			client:  fake.NewClientBuilder().WithRuntimeObjects(config).Build(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := NewAutoNodeSizeConfig()
			ctx := context.Background()

			r := Reconciler{
				Client: test.client,
			}

			err := config.Remove(&r, ctx)
			if err != nil {
				t.Error("Remove could not delete config properly")
			}

			key := types.NamespacedName{
				Name: configName,
			}
			var c mcv1.KubeletConfig

			err = r.Get(ctx, key, &c)
			if err == test.wantErr {
				t.Error("Could not verify config removal")
			}

		})
	}

}
