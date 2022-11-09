package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcofake "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

func TestMachineConfigReconciler(t *testing.T) {
	fakeAro := func(objects ...kruntime.Object) *arofake.Clientset {
		return arofake.NewSimpleClientset(objects...)
	}
	fakeMco := func(objects ...kruntime.Object) *mcofake.Clientset {
		return mcofake.NewSimpleClientset(objects...)
	}
	fakeDh := func(controller *gomock.Controller) *mock_dynamichelper.MockInterface {
		return mock_dynamichelper.NewMockInterface(controller)
	}

	tests := []struct {
		name    string
		arocli  *arofake.Clientset
		mcocli  *mcofake.Clientset
		mocks   func(mdh *mock_dynamichelper.MockInterface)
		request ctrl.Request
		wantErr bool
	}{
		{
			name:    "no cluster",
			arocli:  fakeAro(),
			mcocli:  fakeMco(),
			mocks:   func(mdh *mock_dynamichelper.MockInterface) {},
			request: ctrl.Request{},
			wantErr: true,
		},
		{
			name: "controller disabled",
			arocli: fakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: "false",
						},
					},
				},
			),
			mcocli:  fakeMco(),
			mocks:   func(mdh *mock_dynamichelper.MockInterface) {},
			request: ctrl.Request{},
			wantErr: false,
		},
		{
			name: "no MachineConfigPool for MachineConfig does nothing",
			arocli: fakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: "true",
						},
					},
				},
			),
			mcocli: fakeMco(),
			mocks:  func(mdh *mock_dynamichelper.MockInterface) {},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantErr: false,
		},
		{
			name: "valid MachineConfigPool for MachineConfig reconciles MachineConfig",
			arocli: fakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: "true",
						},
					},
				},
			),
			mcocli: fakeMco(
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			),
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Times(1)
			},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mdh := fakeDh(controller)
			tt.mocks(mdh)

			r := &MachineConfigReconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				arocli: tt.arocli,
				mcocli: tt.mcocli,
				dh:     mdh,
			}

			_, err := r.Reconcile(context.Background(), tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("MachineConfigReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
