package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestRBAC(t *testing.T) {
	enabledCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.RbacEnabled: operator.FlagTrue,
			},
		},
	}

	testCases := []struct {
		desc                     string
		objects                  []client.Object
		wantCreated              map[string]int
		wantUpdated              map[string]int
		wantSREReadOnlyResources []string
		wantErr                  string
	}{
		{
			desc:    "no cluster obj fails",
			objects: []client.Object{},
			wantErr: "clusters.aro.openshift.io \"cluster\" not found",
		},
		{
			desc: "disabled",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: arov1alpha1.SingletonClusterName,
					},
				},
			},
			wantCreated: map[string]int{},
		},
		{
			desc:    "enabled",
			objects: []client.Object{enabledCluster.DeepCopy()},
			wantCreated: map[string]int{
				"ClusterRole//system:aro-diagnostics":        1,
				"ClusterRoleBinding//system:aro-diagnostics": 1,
				"ClusterRole//system:aro-sre":                1,
				"ClusterRoleBinding//system:aro-sre":         1,
			},
			wantSREReadOnlyResources: []string{
				"validatingadmissionpolicies",
				"validatingadmissionpolicybindings",
			},
		},
		{
			desc: "enabled, existing system:aro-sre without validating admission policy access is updated",
			objects: []client.Object{
				enabledCluster.DeepCopy(),
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "system:aro-sre",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"admissionregistration.k8s.io"},
							Resources: []string{"mutatingwebhookconfigurations", "validatingwebhookconfigurations"},
							Verbs:     []string{"get", "list", "watch"},
						},
					},
				},
			},
			wantCreated: map[string]int{
				"ClusterRole//system:aro-diagnostics":        1,
				"ClusterRoleBinding//system:aro-diagnostics": 1,
				"ClusterRoleBinding//system:aro-sre":         1,
			},
			wantUpdated: map[string]int{
				"ClusterRole//system:aro-sre": 1,
			},
			wantSREReadOnlyResources: []string{
				"validatingadmissionpolicies",
				"validatingadmissionpolicybindings",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := require.New(t)
			_, log := testlog.LogForTesting(t)
			createTally := map[string]int{}
			updateTally := map[string]int{}

			clientBuilder := testclienthelper.NewAROFakeClientBuilder(tC.objects...)

			cl := testclienthelper.NewHookingClient(clientBuilder.Build()).
				WithPostCreateHook(testclienthelper.TallyCountsAndKey(createTally))
			cl.WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))
			ch := clienthelper.NewWithClient(log, cl)

			c := NewReconciler(log, ch)

			_, err := c.Reconcile(t.Context(), reconcile.Request{})
			if tC.wantErr == "" {
				r.NoError(err)
			} else {
				r.EqualError(err, tC.wantErr)
			}

			diff, _ := testclienthelper.CompareTally(tC.wantCreated, createTally)
			r.Empty(diff)

			diff, _ = testclienthelper.CompareTally(tC.wantUpdated, updateTally)
			r.Empty(diff)

			if len(tC.wantSREReadOnlyResources) > 0 {
				role := &rbacv1.ClusterRole{}
				r.NoError(cl.Get(t.Context(), types.NamespacedName{Name: "system:aro-sre"}, role))

				for _, resource := range tC.wantSREReadOnlyResources {
					var verbs []string
					for _, rule := range role.Rules {
						if slices.Contains(rule.APIGroups, "admissionregistration.k8s.io") &&
							slices.Contains(rule.Resources, resource) {
							verbs = append(verbs, rule.Verbs...)
						}
					}
					r.ElementsMatch([]string{"get", "list", "watch"}, verbs, resource)
				}
			}
		})
	}
}
