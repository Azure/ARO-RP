package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

func TestSystemreservedEnsure(t *testing.T) {
	tests := []struct {
		name string
		mcp  *mcv1.MachineConfigPool
	}{
		{
			name: "first time create",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker",
				},
			},
		},
		{
			name: "label already exists",
			mcp: &mcv1.MachineConfigPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "worker",
					Labels: map[string]string{labelName: labelValue},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			clientFake := fake.NewClientBuilder().WithObjects(tt.mcp).Build()

			mdh := mock_dynamichelper.NewMockInterface(controller)
			sr := &systemreserved{
				dh:     mdh,
				client: clientFake,
				log:    utillog.GetLogger(),
			}

			mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil)

			err := sr.Ensure(context.Background())
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
		})
	}
}

func TestKubeletConfig(t *testing.T) {
	sr := &systemreserved{}
	got, err := sr.kubeletConfig()
	if err != nil {
		t.Errorf("systemreserved.kubeletConfig() error = %v", err)
		return
	}
	want := &mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aro-limits",
			Labels: map[string]string{
				"aro.openshift.io/limits": "",
			},
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

	if !reflect.DeepEqual(got, want) {
		t.Errorf("systemreserved.kubeletConfig() = %v, want %v", got, want)
		t.Error(cmp.Diff(got, want))
	}
}
