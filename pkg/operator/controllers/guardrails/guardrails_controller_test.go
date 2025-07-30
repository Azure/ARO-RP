package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
)

func TestGuardRailsReconciler(t *testing.T) {
	tests := []struct {
		name          string
		mocks         func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags         arov1alpha1.OperatorFlags
		cleanupNeeded bool
		// errors
		wantErr string
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
			wantErr: "failed delete",
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
			deployer := mock_deployer.NewMockDeployer(controller)
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(cluster)
			log := logrus.NewEntry(logrus.StandardLogger())

			if tt.mocks != nil {
				tt.mocks(deployer, cluster)
			}

			r := &Reconciler{
				log:               log,
				deployer:          deployer,
				ch:                clienthelper.NewWithClient(log, clientBuilder.Build()),
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

func TestGuardrailsTemplateReconcilation(t *testing.T) {
	tests := []struct {
		name          string
		mocks         func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags         arov1alpha1.OperatorFlags
		cleanupNeeded bool
		check         func(context.Context, clienthelper.Interface) error
		// errors
		wantErr string
	}{
		{
			name: "enabled arodenyprivilegednamespace",
			flags: arov1alpha1.OperatorFlags{
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",

				operator.GuardrailsPolicyPrivNamespaceDenyEnforcement: operator.GuardrailsPolicyDryrun,
				operator.GuardrailsPolicyPrivNamespaceDenyManaged:     operator.FlagTrue,
			},
			check: func(ctx context.Context, i clienthelper.Interface) error {
				// Check we have created the templates correctly
				l := &unstructured.UnstructuredList{}
				l.SetGroupVersionKind(schema.GroupVersionKind{Group: "templates.gatekeeper.sh", Version: "v1", Kind: "ConstraintTemplate"})

				foundWantedTemplate := false
				foundWantedConstraint := false

				err := i.List(ctx, l)
				if err != nil {
					return err
				}

				for _, k := range l.Items {
					if k.GroupVersionKind().String() != "templates.gatekeeper.sh/v1, Kind=ConstraintTemplate" {
						return fmt.Errorf("found wrong gvk: %s", l.GroupVersionKind().String())
					}
					if k.GetName() == "arodenyprivilegednamespace" {
						foundWantedTemplate = true
					}
				}

				if !foundWantedTemplate {
					return errors.New("did not find arodenyprivilegednamespace constraint template")
				}

				// Check that we have enabled the specified constraint
				// Check we have created the templates correctly
				lc := &unstructured.UnstructuredList{}
				lc.SetGroupVersionKind(schema.GroupVersionKind{Group: "constraints.gatekeeper.sh", Version: "v1beta1", Kind: "ARODenyPrivilegedNamespace"})

				err = i.List(ctx, lc)
				if err != nil {
					return err
				}

				for _, k := range lc.Items {
					if k.GroupVersionKind().String() != "constraints.gatekeeper.sh/v1beta1, Kind=ARODenyPrivilegedNamespace" {
						return fmt.Errorf("found wrong gvk: %s", l.GroupVersionKind().String())
					}
					if k.GetName() == "aro-privileged-namespace-deny" {
						foundWantedConstraint = true
					}
				}

				if !foundWantedConstraint {
					return errors.New("did not find aro-privileged-namespace-deny constraint")
				}

				return nil
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
			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(cluster)

			log := logrus.NewEntry(logrus.StandardLogger())

			// The Guardrails deployment is always ready in this test
			md := mock_deployer.NewMockDeployer(controller)
			md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
			md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
			md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)

			cl := testclienthelper.NewHookingClient(clientBuilder.Build())

			// Mark constraint templates as ready
			cl = cl.WithPreCreateHook(func(obj client.Object) error {
				if strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) == "constrainttemplate" {
					o := obj.(*unstructured.Unstructured)
					o.Object["status"] = map[string]any{
						"created": true,
					}
				}
				return nil
			})

			ch := clienthelper.NewWithClient(log, cl)

			deployerPolicyTemplate := deployer.NewDeployer(log, cl, gkPolicyTemplates, gkTemplatePath)

			r := &Reconciler{
				log: log,

				deployer:          md,
				ch:                ch,
				gkPolicyTemplate:  deployerPolicyTemplate,
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

			err = tt.check(context.Background(), ch)
			if err != nil {
				t.Errorf("failed validation: %v", err)
			}
		})
	}
}

func TestGuardrailsUpdate(t *testing.T) {
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
				operator.GuardrailsEnabled:       operator.FlagTrue,
				operator.GuardrailsDeployManaged: operator.FlagTrue,
				controllerPullSpec:               "wonderfulPullspec",

				operator.GuardrailsPolicyPrivNamespaceDenyEnforcement: operator.GuardrailsPolicyDryrun,
				operator.GuardrailsPolicyPrivNamespaceDenyManaged:     operator.FlagTrue,
			},
			ACRDomain: "acrtest.example.com",
		},
	}
	clientBuilder := ctrlfake.NewClientBuilder().WithObjects(cluster)

	log := logrus.NewEntry(logrus.StandardLogger())

	// The Guardrails deployment is always ready in this test
	md := mock_deployer.NewMockDeployer(controller)
	md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).AnyTimes().Return(nil)
	md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").AnyTimes().Return(true, nil)
	md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").AnyTimes().Return(true, nil)

	cl := testclienthelper.NewHookingClient(clientBuilder.Build())

	// Mark constraint templates as ready
	cl = cl.WithPreCreateHook(func(obj client.Object) error {
		if strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) == "constrainttemplate" {
			o := obj.(*unstructured.Unstructured)
			o.Object["status"] = map[string]any{
				"created": true,
			}
		}
		return nil
	})

	ch := clienthelper.NewWithClient(log, cl)

	deployerPolicyTemplate := deployer.NewDeployer(log, cl, gkPolicyTemplates, gkTemplatePath)

	r := &Reconciler{
		log: log,

		deployer:          md,
		ch:                ch,
		gkPolicyTemplate:  deployerPolicyTemplate,
		readinessTimeout:  0 * time.Second,
		readinessPollTime: 1 * time.Second,
		cleanupNeeded:     false,
	}
	_, err := r.Reconcile(context.Background(), reconcile.Request{})
	if err != nil {
		t.Fatal(err)
	}

	// Check we have created the templates correctly
	l := &unstructured.UnstructuredList{}
	l.SetGroupVersionKind(schema.GroupVersionKind{Group: "templates.gatekeeper.sh", Version: "v1", Kind: "ConstraintTemplate"})

	foundWantedTemplate := false
	foundWantedConstraint := false

	err = ch.List(context.Background(), l)
	if err != nil {
		t.Fatal(err)
	}

	for _, k := range l.Items {
		if k.GroupVersionKind().String() != "templates.gatekeeper.sh/v1, Kind=ConstraintTemplate" {
			t.Errorf("found wrong gvk: %s", l.GroupVersionKind().String())
		}
		if k.GetName() == "arodenyprivilegednamespace" {
			foundWantedTemplate = true
		}
	}

	if !foundWantedTemplate {
		t.Error("did not find arodenyprivilegednamespace constraint template")
	}

	// Check that we have enabled the specified constraint
	// Check we have created the templates correctly
	lc := &unstructured.UnstructuredList{}
	lc.SetGroupVersionKind(schema.GroupVersionKind{Group: "constraints.gatekeeper.sh", Version: "v1beta1", Kind: "ARODenyPrivilegedNamespace"})

	err = ch.List(context.Background(), lc)
	if err != nil {
		t.Fatal(err)
	}

	for _, k := range lc.Items {
		if k.GroupVersionKind().String() != "constraints.gatekeeper.sh/v1beta1, Kind=ARODenyPrivilegedNamespace" {
			t.Errorf("found wrong gvk: %s", l.GroupVersionKind().String())
		}
		if k.GetName() == "aro-privileged-namespace-deny" {
			foundWantedConstraint = true

			val, ok, err := unstructured.NestedString(k.Object, "spec", "enforcementAction")
			if !ok || err != nil {
				t.Error("error checking constraint")
			}
			if val != "dryrun" {
				t.Errorf("expected constraint to be '%s', was '%s'", "dryrun", val)
			}
		}
	}

	if !foundWantedConstraint {
		t.Fatal("did not find aro-privileged-namespace-deny constraint")
	}

	// Update the enforcement flag
	cluster.Spec.OperatorFlags[operator.GuardrailsPolicyPrivNamespaceDenyEnforcement] = operator.GuardrailsPolicyDeny

	err = ch.Update(context.Background(), cluster)
	if err != nil {
		t.Errorf("failed updating cluster: %v", err)
	}

	_, err = r.Reconcile(context.Background(), reconcile.Request{})
	if err != nil {
		t.Fatal(err)
	}

	// Check that we have enabled the specified constraint
	// Check we have created the templates correctly
	c := &unstructured.Unstructured{}
	c.SetGroupVersionKind(schema.GroupVersionKind{Group: "constraints.gatekeeper.sh", Version: "v1beta1", Kind: "ARODenyPrivilegedNamespace"})

	err = ch.Get(context.Background(), types.NamespacedName{Name: "aro-privileged-namespace-deny"}, c)
	if err != nil {
		t.Fatal(err)
	}

	val, ok, err := unstructured.NestedString(c.Object, "spec", "enforcementAction")
	if !ok || err != nil {
		t.Error("error checking constraint")
	}

	if val != "deny" {
		t.Errorf("expected constraint to be '%s', was '%s'", "deny", val)
	}
}
