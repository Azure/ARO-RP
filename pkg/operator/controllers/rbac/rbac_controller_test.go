package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestRBAC(t *testing.T) {
	testCases := []struct {
		desc        string
		objects     []client.Object
		wantCreated map[string]int
		wantErr     string
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
			desc: "enabled",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: arov1alpha1.SingletonClusterName,
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.RbacEnabled: operator.FlagTrue,
						},
					},
				},
			},
			wantCreated: map[string]int{
				"ClusterRole//system:aro-diagnostics":        1,
				"ClusterRoleBinding//system:aro-diagnostics": 1,
				"ClusterRole//system:aro-sre":                1,
				"ClusterRoleBinding//system:aro-sre":         1,
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
		})
	}
}
