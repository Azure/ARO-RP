package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestAPIServerIsUp(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		objects []runtime.Object
		wantErr string
	}{
		{
			name:    "not found",
			objects: []runtime.Object{},
			wantErr: `TerminalError: clusteroperators.config.openshift.io "kube-apiserver" not found`,
		},
		{
			name: "not ready",
			objects: []runtime.Object{
				&configv1.ClusterOperator{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-apiserver",
					},
					Status: configv1.ClusterOperatorStatus{
						Conditions: []configv1.ClusterOperatorStatusCondition{
							{
								Type:   configv1.OperatorAvailable,
								Status: configv1.ConditionFalse,
							},
							{
								Type:   configv1.OperatorProgressing,
								Status: configv1.ConditionTrue,
							},
						},
					},
				},
			},
			wantErr: `TransientError: kube-apiserver Available=False, Progressing=True`,
		},
		{
			name: "ready",
			objects: []runtime.Object{
				&configv1.ClusterOperator{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-apiserver",
					},
					Status: configv1.ClusterOperatorStatus{
						Conditions: []configv1.ClusterOperatorStatusCondition{
							{
								Type:   configv1.OperatorAvailable,
								Status: configv1.ConditionTrue,
							},
							{
								Type:   configv1.OperatorProgressing,
								Status: configv1.ConditionFalse,
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_, log := testlog.New()

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
			)

			err := EnsureAPIServerIsUp(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}
		})
	}
}
