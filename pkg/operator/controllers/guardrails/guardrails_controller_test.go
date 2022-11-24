package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
)

func TestGuardRailsReconciler(t *testing.T) {
	tests := []struct {
		name  string
		mocks func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags arov1alpha1.OperatorFlags
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
				controllerEnabled:            "true",
				controllerManaged:            "true",
				controllerPullSpec:           "wonderfulPullspec",
				controllerNamespace:          "wonderful-namespace",
				controllerManagerRequestsCPU: "10m",
				controllerManagerLimitCPU:    "100m",
				controllerManagerRequestsMem: "256Mi",
				controllerManagerLimitMem:    "512Mi",
				controllerAuditRequestsCPU:   "10m",
				controllerAuditLimitCPU:      "100m",
				controllerAuditRequestsMem:   "256Mi",
				controllerAuditLimitMem:      "512Mi",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "wonderfulPullspec",
					Namespace:                      "wonderful-namespace",
					ManagerRequestsCPU:             "10m",
					ManagerLimitCPU:                "100m",
					ManagerRequestsMem:             "256Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "10m",
					AuditLimitCPU:                  "100m",
					AuditRequestsMem:               "256Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "wonderful-namespace", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "wonderful-namespace", "gatekeeper-controller-manager").Return(true, nil)
			},
		},
		{
			name: "managed, no pullspec & namespace (uses default)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "quay.io/jeyuan/gatekeeper",
					Namespace:                      "openshift-azure-guardrails",
					ManagerRequestsCPU:             "100m",
					ManagerLimitCPU:                "1000m",
					ManagerRequestsMem:             "256Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "100m",
					AuditLimitCPU:                  "1000m",
					AuditRequestsMem:               "256Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
		},
		{
			name: "managed, GuardRails does not become ready",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "wonderfulPullspec",
					Namespace:                      "openshift-azure-guardrails",
					ManagerRequestsCPU:             "100m",
					ManagerLimitCPU:                "1000m",
					ManagerRequestsMem:             "256Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "100m",
					AuditLimitCPU:                  "1000m",
					AuditRequestsMem:               "256Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: "GateKeeper deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, CreateOrUpdate() fails",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled:  "true",
				controllerManaged:  "true",
				controllerPullSpec: "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.AssignableToTypeOf(&config.GuardRailsDeploymentConfig{})).Return(errors.New("failed ensure"))
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
			deployer := mock_deployer.NewMockDeployer(controller)

			if tt.mocks != nil {
				tt.mocks(deployer, cluster)
			}

			r := &Reconciler{
				arocli:        arocli,
				kubernetescli: kubecli,
				deployer:      deployer,
				// gkPolicyTemplate:   mock_deployer.NewMockDeployer(controller),
				readinessTimeout:  0 * time.Second,
				readinessPollTime: 1 * time.Second,
			}
			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("got error '%v', wanted error '%v'", err, tt.wantErr)
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("did not get an error, but wanted error '%v'", tt.wantErr)
			}
		})
	}
}
