package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitAroOperatorHeartbeat(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		objects        []client.Object
		expectedGauges []expectedMetric
		hooks          func(hc *testclienthelper.HookingClient)
		wantErr        error
	}{
		{
			name: "happy path",
			objects: []client.Object{
				&appsv1.Deployment{ // not available expected
					ObjectMeta: metav1.ObjectMeta{
						Name:       "aro-operator-master",
						Namespace:  "openshift-azure-operator",
						Generation: 4,
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointerutils.ToPtr(int32(1)),
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						AvailableReplicas:   0,
						UnavailableReplicas: 1,
						UpdatedReplicas:     0,
						ObservedGeneration:  4,
					},
				}, &appsv1.Deployment{ // available expected
					ObjectMeta: metav1.ObjectMeta{
						Name:       "aro-operator-worker",
						Namespace:  "openshift-azure-operator",
						Generation: 4,
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointerutils.ToPtr(int32(1)),
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						AvailableReplicas:   1,
						UnavailableReplicas: 0,
						UpdatedReplicas:     1,
						ObservedGeneration:  4,
					},
				}, &appsv1.Deployment{ // no metric expected - different name
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name3",
						Namespace: "openshift-azure-operator",
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						AvailableReplicas:   2,
						UnavailableReplicas: 0,
					},
				}, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{ // no metric expected -customer
						Name:      "name4",
						Namespace: "customer",
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            2,
						AvailableReplicas:   1,
						UnavailableReplicas: 1,
					},
				},
			},
			expectedGauges: []expectedMetric{
				{
					name:   "arooperator.heartbeat",
					value:  int64(0),
					labels: map[string]string{"name": "aro-operator-master"},
				},
				{
					name:   "arooperator.heartbeat",
					value:  int64(1),
					labels: map[string]string{"name": "aro-operator-worker"},
				},
			},
		},
		{
			name: "list error",
			hooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreListHook(func(obj client.ObjectList, opts *client.ListOptions) error {
					return errors.New("list error")
				})
			},
			wantErr: errListAROOperatorDeployments,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)

			_, log := testlog.New()
			client := testclienthelper.NewHookingClient(fake.
				NewClientBuilder().
				WithObjects(tt.objects...).
				Build())
			ocpclientset := clienthelper.NewWithClient(log, client)

			mon := &Monitor{
				log:          log,
				ocpclientset: ocpclientset,
				m:            m,
				queryLimit:   1,
			}

			if tt.hooks != nil {
				tt.hooks(client)
			}

			for _, gauge := range tt.expectedGauges {
				m.EXPECT().EmitGauge(gauge.name, gauge.value, gauge.labels).Times(1)
			}

			err := mon.emitAroOperatorHeartbeat(ctx)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Wanted %v, got %v", err, tt.wantErr)
			} else if tt.wantErr == nil && err != nil {
				t.Fatal(err)
			}
		})
	}
}
