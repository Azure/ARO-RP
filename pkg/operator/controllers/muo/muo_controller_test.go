package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestMUOReconciler(t *testing.T) {
	tests := []struct {
		name           string
		mocks          func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags          arov1alpha1.OperatorFlags
		clusterVersion string
		// errors
		wantErr string
	}{
		{
			name: "disabled",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagFalse,
				operator.MuoManaged: operator.FlagFalse,
				controllerPullSpec:  "wonderfulPullspec",
			},
		},
		{
			name: "managed",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagTrue,
				controllerPullSpec:  "wonderfulPullspec",
			},
			clusterVersion: "4.10.0",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec: "wonderfulPullspec",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, no pullspec (uses default)",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagTrue,
			},
			clusterVersion: "4.10.0",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec: "acrtest.example.com/app-sre/managed-upgrade-operator:v0.1.1202-g118c178",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, MUO does not become ready",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagTrue,
				controllerPullSpec:  "wonderfulPullspec",
			},
			clusterVersion: "4.11.0",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:                     "wonderfulPullspec",
					SupportsPodSecurityAdmission: true,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: "managed Upgrade Operator deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, could not parse cluster version fails",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagTrue,
				controllerPullSpec:  "wonderfulPullspec",
			},
			wantErr: `could not parse version ""`,
		},
		{
			name: "managed, CreateOrUpdate() fails",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagTrue,
				controllerPullSpec:  "wonderfulPullspec",
			},
			clusterVersion: "4.10.0",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.AssignableToTypeOf(&config.MUODeploymentConfig{})).Return(errors.New("failed ensure"))
			},
			wantErr: "failed ensure",
		},
		{
			name: "managed=false (removal)",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagFalse,
				controllerPullSpec:  "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "managed=false (removal), Remove() fails",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: operator.FlagFalse,
				controllerPullSpec:  "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(errors.New("failed delete"))
			},
			wantErr: "failed delete",
		},
		{
			name: "managed=blank (no action)",
			flags: arov1alpha1.OperatorFlags{
				operator.MuoEnabled: operator.FlagTrue,
				operator.MuoManaged: "",
				controllerPullSpec:  "wonderfulPullspec",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			instance := &arov1alpha1.Cluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: "aro.openshift.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: tt.flags,
					ACRDomain:     "acrtest.example.com",
				},
			}

			cv := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: tt.clusterVersion,
						},
					},
				},
			}

			deployer := mock_deployer.NewMockDeployer(controller)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(instance, cv)

			if tt.mocks != nil {
				tt.mocks(deployer, instance)
			}

			r := &Reconciler{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				deployer:          deployer,
				client:            clientBuilder.Build(),
				readinessTimeout:  0 * time.Second,
				readinessPollTime: 1 * time.Second,
			}
			_, err := r.Reconcile(ctx, reconcile.Request{})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
