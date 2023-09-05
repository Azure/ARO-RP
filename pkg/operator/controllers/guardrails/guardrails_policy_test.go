package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	mock_deployer "github.com/Azure/ARO-RP/pkg/util/mocks/deployer"
	testdh "github.com/Azure/ARO-RP/test/util/dynamichelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

//go:embed teststaticresources/gktemplates
var gkPolicyTemplatesTest embed.FS

//go:embed teststaticresources/gkconstraints
var gkPolicyConstraintsTest embed.FS

func TestGuardRailsReconcilerPolicies(t *testing.T) {
	tests := []struct {
		name              string
		mocks             func(*mock_deployer.MockDeployer, *arov1alpha1.Cluster)
		flags             arov1alpha1.OperatorFlags
		timeoutOnTemplate bool
		wantCreated       map[string]int
		wantUpdated       map[string]int
		// wantDeleted includes attempts at deleting (the underlying object
		// might not exist)
		wantDeleted map[string]int
		wantErr     string
		postcheck   func(dynamichelper.Interface) error
	}{
		{
			name: "disabled",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "false",
				controllerManaged: "false",
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
		},
		{
			name: "managed, GuardRails does not become ready, no policy activity",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
			wantErr:     "GateKeeper deployment timed out on Ready: timed out waiting for the condition",
		},
		{
			name: "managed, template times out",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
			timeoutOnTemplate: true,
			wantCreated: map[string]int{
				"ConstraintTemplate//arodenymachineconfig": 1,
			},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
			wantErr:     "GateKeeper ConstraintTemplates timed out on creation: waiting on [arodenymachineconfig]",
		},
		{
			name: "managed, none enabled",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
			wantCreated: map[string]int{
				"ConstraintTemplate//arodenymachineconfig": 1,
			},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{
				"ARODenyMachineConfig//aro-machine-config-deny": 1,
			},
		},
		{
			name: "managed, policy enabled, not enforced",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
				"aro.guardrails.policies.aro-machine-config-deny.managed": "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
			wantCreated: map[string]int{
				"ARODenyMachineConfig//aro-machine-config-deny": 1,
				"ConstraintTemplate//arodenymachineconfig":      1,
			},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
			postcheck: func(i dynamichelper.Interface) error {
				o := &unstructured.Unstructured{}
				o.SetGroupVersionKind(schema.GroupVersionKind{Group: "constraints.gatekeeper.sh", Version: "v1beta1", Kind: "ARODenyMachineConfig"})
				err := i.GetOne(context.Background(), types.NamespacedName{Name: "aro-machine-config-deny"}, o)
				if err != nil {
					return err
				}

				r, found, err := unstructured.NestedString(o.Object, "spec", "enforcementAction")
				if !found {
					return errors.New("invalid constraint")
				}
				if r != "dryrun" {
					return fmt.Errorf("wrong enforcementaction: %s", r)
				}
				return err
			},
		},
		{
			name: "managed, policy enabled, enforced",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
				"aro.guardrails.policies.aro-machine-config-deny.managed":     "true",
				"aro.guardrails.policies.aro-machine-config-deny.enforcement": "deny",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
			wantCreated: map[string]int{
				"ARODenyMachineConfig//aro-machine-config-deny": 1,
				"ConstraintTemplate//arodenymachineconfig":      1,
			},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
			postcheck: func(i dynamichelper.Interface) error {
				o := &unstructured.Unstructured{}
				o.SetGroupVersionKind(schema.GroupVersionKind{Group: "constraints.gatekeeper.sh", Version: "v1beta1", Kind: "ARODenyMachineConfig"})
				err := i.GetOne(context.Background(), types.NamespacedName{Name: "aro-machine-config-deny"}, o)
				if err != nil {
					return err
				}

				r, found, err := unstructured.NestedString(o.Object, "spec", "enforcementAction")
				if !found {
					return errors.New("invalid constraint")
				}
				if r != "deny" {
					return fmt.Errorf("wrong enforcementaction: %s", r)
				}
				return err
			},
		},
		{
			name: "managed, policy enabled, invalid enforcement",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
				"aro.guardrails.policies.aro-machine-config-deny.managed":     "true",
				"aro.guardrails.policies.aro-machine-config-deny.enforcement": "true",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().CreateOrUpdate(gomock.Any(), cluster, gomock.Any()).Return(nil)
				md.EXPECT().IsReady(gomock.Any(), "gatekeeper-system", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-audit").Return(true, nil)
				md.EXPECT().IsReady(gomock.Any(), "openshift-azure-guardrails", "gatekeeper-controller-manager").Return(true, nil)
			},
			wantCreated: map[string]int{
				"ARODenyMachineConfig//aro-machine-config-deny": 1,
				"ConstraintTemplate//arodenymachineconfig":      1,
			},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
			postcheck: func(i dynamichelper.Interface) error {
				o := &unstructured.Unstructured{}
				o.SetGroupVersionKind(schema.GroupVersionKind{Group: "constraints.gatekeeper.sh", Version: "v1beta1", Kind: "ARODenyMachineConfig"})
				err := i.GetOne(context.Background(), types.NamespacedName{Name: "aro-machine-config-deny"}, o)
				if err != nil {
					return err
				}

				r, found, err := unstructured.NestedString(o.Object, "spec", "enforcementAction")
				if !found {
					return errors.New("invalid constraint")
				}
				// will fall back to dryrun if invalid
				if r != "dryrun" {
					return fmt.Errorf("wrong enforcementaction: %s", r)
				}
				return err
			},
		},
		{
			name: "managed=false (removal)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "false",
			},
			mocks: func(md *mock_deployer.MockDeployer, cluster *arov1alpha1.Cluster) {
				md.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{
				"ARODenyMachineConfig//aro-machine-config-deny": 1,
				"ConstraintTemplate//arodenymachineconfig":      1,
			},
		},
		{
			name: "managed=blank (no action)",
			flags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "",
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
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

			deployedObjects := map[string]int{}
			deletedObjects := map[string]int{}
			updatedObjects := map[string]int{}

			setReady := func(obj client.Object) error {
				testdh.TallyCountsAndKey(deployedObjects)(obj)

				if !tt.timeoutOnTemplate && obj.GetObjectKind().GroupVersionKind().Kind == "ConstraintTemplate" {
					u := obj.(*unstructured.Unstructured)
					unstructured.SetNestedField(u.Object, true, "Status", "Created")
				}

				return nil
			}

			wrappedClient := testdh.NewRedirectingClient(clientFake).
				WithCreateHook(setReady).
				WithDeleteHook(testdh.TallyCountsAndKey(deletedObjects)).
				WithUpdateHook(testdh.TallyCountsAndKey(updatedObjects))
			dh := dynamichelper.NewWithClient(log, wrappedClient)

			fakeDeployer := mock_deployer.NewMockDeployer(controller)
			fakeTemplates := deployer.NewDeployer(dh, gkPolicyTemplatesTest, "teststaticresources/gktemplates")
			fakePolicies := deployer.NewDeployer(dh, gkPolicyConstraintsTest, "teststaticresources/gkconstraints")

			if tt.mocks != nil {
				tt.mocks(fakeDeployer, cluster)
			}

			r := &Reconciler{
				log:                 log,
				deployer:            fakeDeployer,
				gkPolicyTemplate:    fakeTemplates,
				gkPolicyConstraints: fakePolicies,
				dh:                  dh,
				readinessTimeout:    0 * time.Second,
				readinessPollTime:   1 * time.Second,
			}

			_, err := r.Reconcile(context.Background(), reconcile.Request{})
			for _, v := range deep.Equal(deployedObjects, tt.wantCreated) {
				t.Errorf("created does not match: %s", v)
			}
			for _, v := range deep.Equal(deletedObjects, tt.wantDeleted) {
				t.Errorf("deleted does not match: %s", v)
			}
			for _, v := range deep.Equal(updatedObjects, tt.wantUpdated) {
				t.Errorf("updated does not match: %s", v)
			}
			if tt.postcheck != nil {
				err = tt.postcheck(dh)
				if err != nil {
					t.Errorf("postcheck failed: %s", err)
				}
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
