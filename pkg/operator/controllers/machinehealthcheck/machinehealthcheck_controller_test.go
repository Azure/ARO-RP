package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
)

// Test reconcile function
func TestReconciler(t *testing.T) {
	type test struct {
		name             string
		arocli           aroclient.Interface
		mocks            func(mdh *mock_dynamichelper.MockInterface)
		wantErr          string
		wantRequeueAfter time.Duration
	}

	for _, tt := range []*test{
		{
			name: "Failure to get instance",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some other name",
				},
			}),
			mocks:   func(mdh *mock_dynamichelper.MockInterface) {},
			wantErr: `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "Enabled Feature Flag is false",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(false),
					},
				},
			}),
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().EnsureDeleted(gomock.Any(), "MachineHealthCheck", "openshift-machine-api", "aro-machinehealthcheck").Times(0)
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			wantErr: "",
		},
		{
			name: "Managed Feature Flag is false: ensure mhc is deleted",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(false),
					},
				},
			}),
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().EnsureDeleted(gomock.Any(), "MachineHealthCheck", "openshift-machine-api", "aro-machinehealthcheck").Times(1)
			},
			wantErr: "",
		},
		{
			name: "Managed Feature Flag is false: mhc fails to delete, an error is returned",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(false),
					},
				},
			}),
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().EnsureDeleted(gomock.Any(), "MachineHealthCheck", "openshift-machine-api", "aro-machinehealthcheck").Return(errors.New("Could not delete mhc"))
			},
			wantErr:          "Could not delete mhc",
			wantRequeueAfter: time.Hour,
		},
		{
			name: "Managed Feature Flag is true: dynamic helper ensures resources",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(true),
					},
				},
			}),
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantErr: "",
		},
		{
			name: "When ensuring resources fails, an error is returned",
			arocli: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						enabled: strconv.FormatBool(true),
						managed: strconv.FormatBool(true),
					},
				},
			}),
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(errors.New("failed to ensure"))
			},
			wantErr: "failed to ensure",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mdh := mock_dynamichelper.NewMockInterface(controller)

			tt.mocks(mdh)

			ctx := context.Background()
			r := &Reconciler{
				arocli: tt.arocli,
				dh:     mdh,
			}
			request := ctrl.Request{}
			request.Name = "cluster"

			result, err := r.Reconcile(ctx, request)

			if tt.wantRequeueAfter != result.RequeueAfter {
				t.Errorf("Wanted to requeue after %v but was set to %v", tt.wantRequeueAfter, result.RequeueAfter)
			}

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
