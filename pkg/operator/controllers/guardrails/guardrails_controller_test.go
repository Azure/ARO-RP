package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func clusterVersionForTest(version string) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{State: configv1.CompletedUpdate, Version: version},
			},
		},
	}
}

func TestGuardRailsReconcilerGatekeeper(t *testing.T) {
	cv := clusterVersionForTest("4.16.0")

	tests := []struct {
		name          string
		mocks         func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		dhMocks       func(*mock_dynamichelper.MockInterface)
		flags         arov1alpha1.OperatorFlags
		cleanupNeeded bool
		wantErr       string
	}{
		{
			name: "disabled",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagFalse,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
				controllerPullSpec:               "wonderfulPullspec",
			},
		},
		{
			name: "managed",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",
				controllerNamespace:              "wonderful-namespace",
				controllerManagerRequestsCPU:     "10m",
				controllerManagerLimitCPU:        "100m",
				controllerManagerRequestsMem:     "512Mi",
				controllerManagerLimitMem:        "512Mi",
				controllerAuditRequestsCPU:       "10m",
				controllerAuditLimitCPU:          "100m",
				controllerAuditRequestsMem:       "512Mi",
				controllerAuditLimitMem:          "512Mi",
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
				md.EXPECT().IsReady(gomock.Any(), "wonderful-namespace", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "wonderful-namespace", "gatekeeper-controller-manager").Return(true, nil)
			},
		},
		{
			name: "managed, no pullspec & namespace (uses default)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				expectedConfig := &config.GuardRailsDeploymentConfig{
					Pullspec:                       "acrtest.example.com/gatekeeper:v3.19.2",
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
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
		},
		{
			name: "managed, GuardRails does not become ready",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",
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
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: "GateKeeper deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, CreateOrUpdate() fails",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.AssignableToTypeOf(&config.GuardRailsDeploymentConfig{})).Return(errors.New("failed ensure"))
			},
			wantErr: "failed ensure",
		},
		{
			name: "managed=false (removal)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
				controllerPullSpec:               "wonderfulPullspec",
			},
			cleanupNeeded: true,
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
		},
		{
			name: "managed=false (removal), Remove() fails",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
				controllerPullSpec:               "wonderfulPullspec",
			},
			cleanupNeeded: true,
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(errors.New("failed delete"))
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: "failed to remove Gatekeeper deployment: failed delete",
		},
		{
			name: "managed=blank (no action)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: "",
				controllerPullSpec:               "wonderfulPullspec",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

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
			dep := mock_deployer.NewMockDeployer(controller)
			dh := mock_dynamichelper.NewMockInterface(controller)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(cluster, cv)

			if tt.mocks != nil {
				tt.mocks(dep, cluster)
			}
			if tt.dhMocks != nil {
				tt.dhMocks(dh)
			}

			r := &Reconciler{
				log:               logrus.NewEntry(logrus.StandardLogger()),
				deployer:          dep,
				dh:                dh,
				client:            clientBuilder.Build(),
				readinessTimeout:  0 * time.Second,
				readinessPollTime: 1 * time.Second,
				cleanupNeeded:     tt.cleanupNeeded,
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

func TestReconcileVAP(t *testing.T) {
	cv := clusterVersionForTest("4.17.0")

	tests := []struct {
		name          string
		flags         arov1alpha1.OperatorFlags
		depMocks      func(*mock_deployer.MockDeployer)
		dhMocks       func(*mock_dynamichelper.MockInterface)
		cleanupNeeded bool
		wantErr       string
	}{
		{
			name: "VAP: disabled",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagFalse,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
			},
		},
		{
			name: "VAP: managed=true, deploys VAP policies",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:                            operator.FlagTrue,
				operator.GuardrailsDeployManaged:                      operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyManaged:           operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyEnforcement:       operator.GuardrailsPolicyDeny,
				operator.GuardrailsPolicyMachineConfigDenyManaged:     operator.FlagTrue,
				operator.GuardrailsPolicyMachineConfigDenyEnforcement: operator.GuardrailsPolicyDryrun,
				operator.GuardrailsPolicyPrivNamespaceDenyManaged:     operator.FlagTrue,
				operator.GuardrailsPolicyPrivNamespaceDenyEnforcement: operator.GuardrailsPolicyWarn,
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				// 3 policies + 3 bindings = 6 Ensure calls (server-side apply)
				dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(6)
			},
		},
		{
			name: "VAP: managed=true with gatekeeper migration (upgrade from pre-4.17)",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:                            operator.FlagTrue,
				operator.GuardrailsDeployManaged:                      operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyManaged:           operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyEnforcement:       operator.GuardrailsPolicyDeny,
				operator.GuardrailsPolicyMachineConfigDenyManaged:     operator.FlagTrue,
				operator.GuardrailsPolicyMachineConfigDenyEnforcement: operator.GuardrailsPolicyDryrun,
				operator.GuardrailsPolicyPrivNamespaceDenyManaged:     operator.FlagTrue,
				operator.GuardrailsPolicyPrivNamespaceDenyEnforcement: operator.GuardrailsPolicyWarn,
			},
			cleanupNeeded: true,
			depMocks: func(md *mock_deployer.MockDeployer) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				// cleanupGatekeeper: removePolicy calls EnsureDeletedGVR for GK constraints
				// then deployVAP: 3 Ensure (policy) + 3 EnsureDeletedGVR (binding delete) + 3 Ensure (binding recreate)
				dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(6)
			},
		},
		{
			name: "VAP: managed=false, removes all policies",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagFalse,
			},
			dhMocks: func(dh *mock_dynamichelper.MockInterface) {
				// 3 policies × (binding delete + policy delete) = 6 deletions
				dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(6)
			},
		},
		{
			name: "VAP: managed=blank, no action",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

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
				},
			}

			dep := mock_deployer.NewMockDeployer(controller)
			dh := mock_dynamichelper.NewMockInterface(controller)
			if tt.depMocks != nil {
				tt.depMocks(dep)
			}
			if tt.dhMocks != nil {
				tt.dhMocks(dh)
			}

			r := &Reconciler{
				log:           logrus.NewEntry(logrus.StandardLogger()),
				deployer:      dep,
				client:        ctrlfake.NewClientBuilder().WithObjects(cluster, cv).Build(),
				dh:            dh,
				cleanupNeeded: tt.cleanupNeeded,
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

func TestVapValidationAction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"deny", "Deny"},
		{"Deny", "Deny"},
		{"warn", "Warn"},
		{"Warn", "Warn"},
		{"dryrun", "Audit"},
		{"DryRun", "Audit"},
		{"", "Warn"},
		{"unknown", "Warn"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := vapValidationAction(tt.input)
			if got != tt.want {
				t.Errorf("vapValidationAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeployVAPUsesLatestClusterState(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	cluster := &arov1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "aro.openshift.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.GuardrailsPolicyMachineDenyManaged:       operator.FlagTrue,
				operator.GuardrailsPolicyMachineDenyEnforcement:   operator.GuardrailsPolicyDeny,
				operator.GuardrailsPolicyMachineConfigDenyManaged: operator.FlagFalse,
				operator.GuardrailsPolicyPrivNamespaceDenyManaged: operator.FlagFalse,
			},
		},
	}

	var ensured []string
	var deleted []string

	dh := mock_dynamichelper.NewMockInterface(controller)
	dh.EXPECT().Ensure(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, objs ...any) error {
		for _, obj := range objs {
			o, ok := obj.(interface {
				GetObjectKind() schema.ObjectKind
				GetName() string
			})
			if !ok {
				continue
			}
			ensured = append(ensured, o.GetObjectKind().GroupVersionKind().Kind+"/"+o.GetName())
		}
		return nil
	}).AnyTimes()
	dh.EXPECT().EnsureDeletedGVR(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, groupKind, namespace, name, optionalVersion string) error {
		deleted = append(deleted, groupKind+"/"+name)
		return nil
	}).AnyTimes()
	dh.EXPECT().Refresh().Return(nil).AnyTimes()
	dh.EXPECT().EnsureDeleted(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	dh.EXPECT().IsConstraintTemplateReady(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()

	r := &Reconciler{
		log:    logrus.NewEntry(logrus.StandardLogger()),
		client: ctrlfake.NewClientBuilder().WithObjects(cluster).Build(),
		dh:     dh,
	}

	if err := r.deployVAP(context.Background()); err != nil {
		t.Fatalf("first deployVAP() returned error: %v", err)
	}

	if !slices.Contains(ensured, "ValidatingAdmissionPolicy/aro-machines-deny") {
		t.Fatalf("expected aro-machines-deny policy to be ensured, got %v", ensured)
	}
	if !slices.Contains(ensured, "ValidatingAdmissionPolicyBinding/aro-machines-deny-binding") {
		t.Fatalf("expected aro-machines-deny binding to be ensured, got %v", ensured)
	}

	ensured = nil
	deleted = nil

	latest := &arov1alpha1.Cluster{}
	if err := r.client.Get(context.Background(), client.ObjectKey{Name: arov1alpha1.SingletonClusterName}, latest); err != nil {
		t.Fatalf("failed to get cluster from fake client: %v", err)
	}
	latest.Spec.OperatorFlags[operator.GuardrailsPolicyMachineDenyManaged] = operator.FlagFalse
	latest.Spec.OperatorFlags[operator.GuardrailsPolicyPrivNamespaceDenyManaged] = operator.FlagTrue
	latest.Spec.OperatorFlags[operator.GuardrailsPolicyPrivNamespaceDenyEnforcement] = operator.GuardrailsPolicyWarn
	if err := r.client.Update(context.Background(), latest); err != nil {
		t.Fatalf("failed to update cluster in fake client: %v", err)
	}

	if err := r.deployVAP(context.Background()); err != nil {
		t.Fatalf("second deployVAP() returned error: %v", err)
	}

	if !slices.Contains(deleted, "ValidatingAdmissionPolicyBinding.admissionregistration.k8s.io/aro-machines-deny-binding") {
		t.Fatalf("expected aro-machines-deny binding to be deleted after flag change, got %v", deleted)
	}
	if !slices.Contains(deleted, "ValidatingAdmissionPolicy.admissionregistration.k8s.io/aro-machines-deny") {
		t.Fatalf("expected aro-machines-deny policy to be deleted after flag change, got %v", deleted)
	}
	if !slices.Contains(ensured, "ValidatingAdmissionPolicy/aro-privileged-namespace-deny") {
		t.Fatalf("expected aro-privileged-namespace-deny policy to be ensured after flag change, got %v", ensured)
	}
	if !slices.Contains(ensured, "ValidatingAdmissionPolicyBinding/aro-privileged-namespace-deny-binding") {
		t.Fatalf("expected aro-privileged-namespace-deny binding to be ensured after flag change, got %v", ensured)
	}
	if slices.Contains(ensured, "ValidatingAdmissionPolicy/aro-machines-deny") {
		t.Fatalf("did not expect aro-machines-deny policy to be re-ensured after it was disabled, got %v", ensured)
	}
}
