package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
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
				controllerManagerRequestsMem: "512Mi",
				controllerManagerLimitMem:    "512Mi",
				controllerAuditRequestsCPU:   "10m",
				controllerAuditLimitCPU:      "100m",
				controllerAuditRequestsMem:   "512Mi",
				controllerAuditLimitMem:      "512Mi",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "wonderfulPullspec",
					Namespace:                      "wonderful-namespace",
					ManagerRequestsCPU:             "10m",
					ManagerLimitCPU:                "100m",
					ManagerRequestsMem:             "512Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "10m",
					AuditLimitCPU:                  "100m",
					AuditRequestsMem:               "512Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
					RoleSCCResourceName:            "restricted-v2",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
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
					Pullspec:                       "acrtest.example.com/gatekeeper:v3.11.1",
					Namespace:                      "openshift-azure-guardrails",
					ManagerRequestsCPU:             "100m",
					ManagerLimitCPU:                "1000m",
					ManagerRequestsMem:             "512Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "100m",
					AuditLimitCPU:                  "1000m",
					AuditRequestsMem:               "512Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
					RoleSCCResourceName:            "restricted-v2",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
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
					ManagerRequestsMem:             "512Mi",
					ManagerLimitMem:                "512Mi",
					AuditRequestsCPU:               "100m",
					AuditLimitCPU:                  "1000m",
					AuditRequestsMem:               "512Mi",
					AuditLimitMem:                  "512Mi",
					ValidatingWebhookTimeout:       "3",
					ValidatingWebhookFailurePolicy: "Ignore",
					MutatingWebhookTimeout:         "1",
					MutatingWebhookFailurePolicy:   "Ignore",
					RoleSCCResourceName:            "restricted-v2",
				}
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, expectedConfig).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
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
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
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
			_, log := testlog.New()

			cluster := &arov1alpha1.Cluster{
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
				Status: configv1.ClusterVersionStatus{History: []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.11.0",
					},
				}},
			}

			clientFake := ctrlfake.NewClientBuilder().
				WithObjects(cluster, cv).
				Build()

			dh := dynamichelper.NewWithClient(log, clientFake)
			deployer := mock_deployer.NewMockDeployer(controller)

			if tt.mocks != nil {
				tt.mocks(deployer, cluster)
			}

			r := &Reconciler{
				log:               log,
				deployer:          deployer,
				dh:                dh,
				readinessTimeout:  0 * time.Second,
				readinessPollTime: 1 * time.Second,
			}
			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
