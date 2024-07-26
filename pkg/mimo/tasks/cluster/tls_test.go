package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestConfigureAPIServerCertificates(t *testing.T) {
	ctx := context.Background()
	clusterUUID := uuid.DefaultGenerator.Generate()

	for _, tt := range []struct {
		name    string
		objects []runtime.Object
		wantErr string
	}{
		{
			name:    "not found",
			objects: []runtime.Object{},
			wantErr: `NonRetryableError: apiserver.config.openshift.io "cluster" not found`,
		},
		{
			name: "secrets referenced",
			objects: []runtime.Object{
				&configv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: configv1.APIServerSpec{},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Domain().AnyTimes().Return("example.com")

			_, log := testlog.New()

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterProperties(clusterUUID, api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Domain: "something.k8s.example.com",
					},
				}),
			)

			err := EnsureAPIServerServingCertificateConfiguration(tc)
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
