package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	mock_muo "github.com/Azure/ARO-RP/pkg/operator/mocks/muo"
)

func TestMUOReconciler(t *testing.T) {
	tests := []struct {
		name  string
		mocks func(*mock_muo.MockDeployer, *arov1alpha1.Cluster)
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
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec: "wonderfulPullspec",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, no pullspec (uses default)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec: "acrtest.example.com/managed-upgrade-operator:aro-b1",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM allowed but pull secret entirely missing",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerAllowOCM: "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM allowed but empty pullsecret",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerAllowOCM: "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {}}",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM allowed but mangled pullsecret",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerAllowOCM: "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			pullsecret: "i'm a little json, short and stout",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: false,
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM connected mode",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerAllowOCM: "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {\"" + pullSecretOCMKey + "\": {\"auth\": \"secret value\"}}}",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: true,
					OCMBaseURL:      "https://api.openshift.com",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, OCM connected mode, custom OCM URL",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:    "true",
				controllerManaged:    "true",
				controllerAllowOCM:   "true",
				controllerOcmBaseURL: "https://example.com",
				controllerPullSpec:   "wonderfulPullspec",
			},
			pullsecret: "{\"auths\": {\"" + pullSecretOCMKey + "\": {\"auth\": \"secret value\"}}}",
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec:        "wonderfulPullspec",
					EnableConnected: true,
					OCMBaseURL:      "https://example.com",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "managed, MUO does not become ready",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.MUODeploymentConfig{
					Pullspec: "wonderfulPullspec",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any()).Return(false, nil)
			},
			wantErr: "Managed Upgrade Operator deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, CreateOrUpdate() fails",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
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
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any()).Return(nil)
			},
		},
		{
			name: "managed=false (removal), Remove() fails",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "false",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_muo.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any()).Return(errors.New("failed delete"))
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
			controller := gomock.NewController(t)
			defer controller.Finish()

			cluster := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: tt.flags,
					ACRDomain:     "acrtest.example.com",
				},
			}
			arocli := arofake.NewSimpleClientset(cluster)
			kubecli := fake.NewSimpleClientset()
			deployer := mock_muo.NewMockDeployer(controller)

			if tt.pullsecret != "" {
				_, err := kubecli.CoreV1().Secrets(pullSecretName.Namespace).Create(context.Background(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pullSecretName.Name,
							Namespace: pullSecretName.Namespace,
						},
						Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(tt.pullsecret)},
					},
					metav1.CreateOptions{})
				if err != nil {
					t.Fatal(err)
				}
			}

			if tt.mocks != nil {
				tt.mocks(deployer, cluster)
			}

			r := &Reconciler{
				arocli:            arocli,
				kubernetescli:     kubecli,
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
