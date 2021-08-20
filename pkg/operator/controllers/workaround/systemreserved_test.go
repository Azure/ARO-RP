package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

func TestSystemreservedEnsure(t *testing.T) {

	kc, err := kubeletConfig()
	if err != nil {
		t.Fail()
	}

	tests := []struct {
		name                         string
		mcocli                       *mcofake.Clientset
		mocker                       func(mdh *mock_dynamichelper.MockInterface)
		machineConfigPoolNeedsUpdate bool
		kubeletConfigNeedsDelete     bool
		wantErr                      bool
	}{
		{
			name: "first time create",
			mcocli: mcofake.NewSimpleClientset(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker",
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "master",
					},
				}),
			machineConfigPoolNeedsUpdate: true,
			mocker: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "nothing to be done",
			mcocli: mcofake.NewSimpleClientset(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "worker",
						Labels: map[string]string{labelName: labelValue},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "master",
						Labels: map[string]string{labelName: labelValue},
					},
				},
			),
			mocker: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "masters needs update",
			mcocli: mcofake.NewSimpleClientset(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "worker",
						Labels: map[string]string{labelName: labelValue},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "master",
					},
				},
				kc,
			),
			machineConfigPoolNeedsUpdate: true,
			kubeletConfigNeedsDelete:     true,
			mocker: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mdh := mock_dynamichelper.NewMockInterface(controller)
			sr := &systemreserved{
				mcocli: tt.mcocli,
				dh:     mdh,
				log:    utillog.GetLogger(),
			}

			var updated, deleted bool
			tt.mcocli.PrependReactor("update", "machineconfigpools", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})
			tt.mcocli.PrependReactor("delete", "kubeletconfigs", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				deleted = true
				return false, nil, nil
			})

			tt.mocker(mdh)
			if err := sr.Ensure(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("systemreserved.Ensure() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.machineConfigPoolNeedsUpdate != updated {
				t.Errorf("systemreserved.Ensure() updated %v, machineConfigPoolNeedsUpdate = %v", updated, tt.machineConfigPoolNeedsUpdate)
			}
			if tt.kubeletConfigNeedsDelete != deleted {
				t.Errorf("KubeletConfigs().Delete deleted %v, kubeletConfigNeedsDelete = %v", deleted, tt.kubeletConfigNeedsDelete)
			}
		})
	}
}

func TestKubeletConfig(t *testing.T) {
	got, err := kubeletConfig()
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
			KubeletConfig: &runtime.RawExtension{
				Raw: []byte(`{"evictionHard":{"imagefs.available":"15%","memory.available":"500Mi","nodefs.available":"10%","nodefs.inodesFree":"5%"},"systemReserved":{"memory":"2000Mi"}}`),
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("systemreserved.kubeletConfig() = %v, want %v", got, want)
		t.Error(cmp.Diff(got, want))
	}
}
