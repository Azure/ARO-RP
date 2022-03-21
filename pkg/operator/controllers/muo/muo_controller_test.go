package muo

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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	mock_muo "github.com/Azure/ARO-RP/pkg/operator/mocks/muo"
)

func TestMUOReconciler(t *testing.T) {
	tests := []struct {
		name  string
		mocks func(*mock_muo.MockDeployer, *arov1alpha1.Cluster)
		// feature flag options
		enabled bool
		managed string
		// errors
		wantErr string
	}{
		{
			name:    "disabled",
			enabled: false,
			managed: "false",
		},
		{
			name:    "managed",
			enabled: true,
			managed: "true",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name:    "managed, MUO does not become ready",
			enabled: true,
			managed: "true",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(false, nil)
			},
			wantErr: "Managed Upgrade Operator deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name:    "managed, CreateOrUpdate() fails",
			enabled: true,
			managed: "true",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster).Return(errors.New("failed ensure"))
			},
			wantErr: "failed ensure",
		},
		{
			name:    "managed=false (removal)",
			enabled: true,
			managed: "false",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any()).Return(nil)
			},
		},
		{
			name:    "managed=false (removal), Remove() fails",
			enabled: true,
			managed: "false",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any()).Return(errors.New("failed delete"))
			},
			wantErr: "failed delete",
		},
		{
			name:    "managed=blank (no action)",
			enabled: true,
			managed: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			cluster := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled:  strconv.FormatBool(tt.enabled),
						controllerPullSpec: "wonderfulPullspec",
					},
				},
			}
			// nil and empty string are valid values, be careful to preserve them
			if tt.managed != "" {
				cluster.Spec.OperatorFlags[controllerManaged] = tt.managed
			}
			arocli := arofake.NewSimpleClientset(cluster)
			deployer := mock_muo.NewMockDeployer(controller)

			if tt.mocks != nil {
				tt.mocks(deployer, cluster)
			}

			r := &Reconciler{
				arocli:            arocli,
				deployer:          deployer,
				readinessTimeout:  0 * time.Second,
				readinessPollTime: 1 * time.Second,
			}
			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			if err == nil && tt.wantErr != "" {
				t.Error(err)
			} else if err != nil {
				if err.Error() != tt.wantErr {
					t.Errorf("wanted '%v', got '%v'", tt.wantErr, err)
				}
			}
		})
	}
}
