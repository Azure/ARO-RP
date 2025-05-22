package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestOperatorFlags(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		objects     []runtime.Object
		wantObjects []runtime.Object
		wantErr     string
	}{
		{
			name:    "not found",
			objects: []runtime.Object{},
			wantErr: `TerminalError: clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "not ready",
			objects: []runtime.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:            arov1alpha1.SingletonClusterName,
						ResourceVersion: "1000",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Cluster",
						APIVersion: arov1alpha1.SchemeGroupVersion.String(),
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							"foo": "bar",
						},
					},
				},
			},
			wantObjects: []runtime.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:            arov1alpha1.SingletonClusterName,
						ResourceVersion: "1001",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Cluster",
						APIVersion: arov1alpha1.SchemeGroupVersion.String(),
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							"foo": "baz",
							"gaz": "data",
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

			ocDoc := &api.OpenShiftClusterDocument{
				ID: "0000",
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						OperatorFlags: api.OperatorFlags{
							"foo": "baz",
							"gaz": "data",
						},
					},
				},
			}

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch), testtasks.WithOpenShiftClusterDocument(ocDoc),
			)

			err := UpdateClusterOperatorFlags(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if len(tt.wantObjects) > 0 {
				for _, i := range tt.wantObjects {
					o, err := scheme.Scheme.New(i.GetObjectKind().GroupVersionKind())
					g.Expect(err).ToNot(HaveOccurred())

					err = ch.GetOne(ctx, client.ObjectKeyFromObject(i.(client.Object)), o)
					g.Expect(err).ToNot(HaveOccurred())

					r := deep.Equal(i, o)
					g.Expect(r).To(BeEmpty())
				}
			}
		})
	}
}
