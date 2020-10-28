package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	fakeconfigclient "github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_workaround "github.com/Azure/ARO-RP/pkg/util/mocks/operator/controllers/workaround"
)

func clusterVersion(ver string) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			Desired: configv1.Update{
				Version: ver,
			},
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: ver,
				},
			},
		},
	}
}

func TestWorkaroundReconciler(t *testing.T) {
	tests := []struct {
		name    string
		want    ctrl.Result
		mocker  func(mw *mock_workaround.MockWorkaround)
		wantErr bool
	}{
		{
			name: "is required",
			mocker: func(mw *mock_workaround.MockWorkaround) {
				c := mw.EXPECT().IsRequired(gomock.Any()).Return(true)
				mw.EXPECT().Ensure(gomock.Any()).After(c).Return(nil)
			},
			want: ctrl.Result{Requeue: true, RequeueAfter: time.Hour},
		},
		{
			name: "is not required",
			mocker: func(mw *mock_workaround.MockWorkaround) {
				c := mw.EXPECT().IsRequired(gomock.Any()).Return(false)
				mw.EXPECT().Remove(gomock.Any()).After(c).Return(nil)
			},
			want: ctrl.Result{Requeue: true, RequeueAfter: time.Hour},
		},
		{
			name: "has error",
			mocker: func(mw *mock_workaround.MockWorkaround) {
				mw.EXPECT().IsRequired(gomock.Any()).Return(true)
				mw.EXPECT().Name().Return("test").AnyTimes()
				mw.EXPECT().Ensure(gomock.Any()).Return(fmt.Errorf("oops"))
			},
			want:    ctrl.Result{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mwa := mock_workaround.NewMockWorkaround(controller)
			r := &WorkaroundReconciler{
				configcli:   fakeconfigclient.NewSimpleClientset(clusterVersion("4.4.10")),
				workarounds: []Workaround{mwa},
				log:         utillog.GetLogger(),
			}
			tt.mocker(mwa)
			got, err := r.Reconcile(reconcile.Request{})
			if (err != nil) != tt.wantErr {
				t.Errorf("WorkaroundReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WorkaroundReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
