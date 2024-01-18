package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestSystemreservedEnsure(t *testing.T) {
	kubeletConfig := func(resourceVersion string) *mcv1.KubeletConfig {
		return &mcv1.KubeletConfig{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KubeletConfig",
				APIVersion: "machineconfiguration.openshift.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "aro-limits",
				Labels: map[string]string{
					"aro.openshift.io/limits": "",
				},
				ResourceVersion: resourceVersion,
			},
			Spec: mcv1.KubeletConfigSpec{
				MachineConfigPoolSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"aro.openshift.io/limits": "",
					},
				},
				KubeletConfig: &kruntime.RawExtension{
					Raw: []byte(`{"evictionHard":{"imagefs.available":"15%","memory.available":"500Mi","nodefs.available":"10%","nodefs.inodesFree":"5%"},"systemReserved":{"memory":"2000Mi"}}`),
				},
			},
		}
	}

	tests := []struct {
		name              string
		mcp               *mcv1.MachineConfigPool
		kc                *mcv1.KubeletConfig
		wantKubeletConfig *mcv1.KubeletConfig
	}{
		{
			name: "first time create KubeletConfig",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker",
				},
			},
			wantKubeletConfig: kubeletConfig("1"),
		},
		{
			name: "label already exists on MCP, but no KubeletConfig",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "worker",
					Labels: map[string]string{labelName: labelValue},
				},
			},
			wantKubeletConfig: kubeletConfig("1"),
		},
		{
			name: "no label on MCP, but KubeletConfig exists",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker",
				},
			},
			kc: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:            kubeletConfigName,
					ResourceVersion: "1",
				},
			},
			wantKubeletConfig: kubeletConfig("2"),
		},
		{
			name: "label and KubeletConfig already exist",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "worker",
					Labels: map[string]string{labelName: labelValue},
				},
			},
			kc: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:            kubeletConfigName,
					ResourceVersion: "1",
				},
			},
			wantKubeletConfig: kubeletConfig("2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.mcp != nil {
				clientBuilder = clientBuilder.WithObjects(tt.mcp)
			}
			if tt.kc != nil {
				clientBuilder = clientBuilder.WithObjects(tt.kc)
			}
			clientFake := clientBuilder.Build()

			sr := &systemreserved{
				client: clientFake,
				log:    utillog.GetLogger(),
			}

			err := sr.Ensure(ctx)
			if err != nil {
				t.Error(err)
			}

			result := &mcv1.MachineConfigPool{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: workerMachineConfigPoolName}, result)
			if err != nil {
				t.Fatal(err)
			}

			if val, ok := result.Labels[labelName]; !ok || val != labelValue {
				t.Error(result.Labels)
			}

			kc := &mcv1.KubeletConfig{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: kubeletConfigName}, kc)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(kc, tt.wantKubeletConfig) {
				t.Error(cmp.Diff(kc, tt.wantKubeletConfig))
			}
		})
	}
}

func TestSystemreservedRemove(t *testing.T) {
	tests := []struct {
		name string
		mcp  *mcv1.MachineConfigPool
		kc   *mcv1.KubeletConfig
	}{
		{
			name: "label is not set, not KubeletConfig",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker",
				},
			},
		},
		{
			name: "label is not set, KubeletConfig exists",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker",
				},
			},
			kc: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:            kubeletConfigName,
					ResourceVersion: "1",
				},
			},
		},
		{
			name: "label is set, but KubeletConfig does not exist",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "worker",
					Labels: map[string]string{labelName: labelValue},
				},
			},
		},
		{
			name: "both label and KubeletConfig set exist",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "worker",
					Labels: map[string]string{labelName: labelValue},
				},
			},
			kc: &mcv1.KubeletConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:            kubeletConfigName,
					ResourceVersion: "1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.mcp != nil {
				clientBuilder = clientBuilder.WithObjects(tt.mcp)
			}

			clientFake := clientBuilder.Build()

			sr := &systemreserved{
				client: clientFake,
				log:    utillog.GetLogger(),
			}

			err := sr.Remove(ctx)
			if err != nil {
				t.Error(err)
			}

			result := &mcv1.MachineConfigPool{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: workerMachineConfigPoolName}, result)
			if err != nil {
				t.Fatal(err)
			}

			if _, ok := result.Labels[labelName]; ok {
				t.Error(result.Labels)
			}

			kc := &mcv1.KubeletConfig{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: kubeletConfigName}, kc)
			if !kerrors.IsNotFound(err) {
				t.Error(err)
			}
		})
	}
}
