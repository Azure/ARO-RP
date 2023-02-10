package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestMUOReconciler(t *testing.T) {
	tests := []struct {
		name  string
		mocks func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags arov1alpha1.OperatorFlags
		// connected MUO -- cluster pullsecret
		pullsecret string
		// errors
		wantErr string
	}{
		{
			name: "disabled",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "false",
				controllerManaged:  "false",
				controllerPullSpec: "wonderfulPullspec",
			},
		},
		{
			name: "managed",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, no pullspec (uses default)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "acrtest.example.com/managed-upgrade-operator:aro-b4",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM allowed but pull secret entirely missing",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:        "true",
				controllerManaged:        "true",
				controllerForceLocalOnly: "false",
				controllerPullSpec:       "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM allowed but empty pullsecret",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:        "true",
				controllerManaged:        "true",
				controllerForceLocalOnly: "false",
				controllerPullSpec:       "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {}}",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM allowed but mangled pullsecret",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:        "true",
				controllerManaged:        "true",
				controllerForceLocalOnly: "false",
				controllerPullSpec:       "wonderfulPullspec",
			},
			pullsecret: "i'm a little json, short and stout",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM connected mode",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:        "true",
				controllerManaged:        "true",
				controllerForceLocalOnly: "false",
				controllerPullSpec:       "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {\"" + pullSecretOCMKey + "\": {\"auth\": \"secret value\"}}}",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: true,
					OCMBaseURL:      "https://api.openshift.com",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM connected mode, custom OCM URL",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:        "true",
				controllerManaged:        "true",
				controllerForceLocalOnly: "false",
				controllerOcmBaseURL:     "https://example.com",
				controllerPullSpec:       "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {\"" + pullSecretOCMKey + "\": {\"auth\": \"secret value\"}}}",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: true,
					OCMBaseURL:      "https://example.com",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, pull secret exists, OCM disabled",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:        "true",
				controllerManaged:        "true",
				controllerForceLocalOnly: "true",
				controllerPullSpec:       "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {\"" + pullSecretOCMKey + "\": {\"auth\": \"secret value\"}}}",
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, MUO does not become ready",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: "managed Upgrade Operator deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, CreateOrUpdate() fails",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.AssignableToTypeOf(&config.MUODeploymentConfig{})).Return(errors.New("failed ensure"))
			},
			wantErr: "failed ensure",
		},
		{
			name: "managed=false (removal)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "false",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "managed=false (removal), Remove() fails",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "false",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(errors.New("failed delete"))
			},
			wantErr: "failed delete",
		},
		{
			name: "managed=blank (no action)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "",
				controllerPullSpec: "wonderfulPullspec",
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
			deployer := mock_deployer.NewMockDeployer(controller)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(instance)

			if tt.pullsecret != "" {
				clientBuilder = clientBuilder.WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pullSecretName.Name,
						Namespace: pullSecretName.Namespace,
					},
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(tt.pullsecret)},
				})
			}

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
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error '%v', wanted error '%v'", err, tt.wantErr)
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("did not get an error, but wanted error '%v'", tt.wantErr)
			}
		})
	}
}
