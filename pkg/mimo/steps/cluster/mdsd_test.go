package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestMDSDRotate(t *testing.T) {
	ctx := context.Background()

	validKey, validCerts, err := utiltls.GenerateKeyAndCertificate("cert", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	encodedKey, err := utilpem.Encode(validKey)
	if err != nil {
		t.Fatal(err)
	}
	encodedCert, err := utilpem.Encode(validCerts[0])
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name    string
		objects []runtime.Object
		check   func(clienthelper.Interface, Gomega)
		wantErr string
	}{
		{
			name:    "secret created (did not exist)",
			objects: []runtime.Object{},
			wantErr: "TerminalError: failed to fetch operator secret object: secrets \"cluster\" not found",
		},
		{
			name: "secret updated",
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster",
						Namespace: "openshift-azure-operator",
					},
					Data: map[string][]byte{"extdata": {'a'}},
				},
			},
			check: func(i clienthelper.Interface, g Gomega) {
				s := &corev1.Secret{}
				err := i.GetOne(ctx, types.NamespacedName{Name: "cluster", Namespace: "openshift-azure-operator"}, s)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(maps.Keys(s.Data)).To(ContainElements("gcscert.pem", "gcskey.pem", "extdata"), "MDSD certs")
				g.Expect(s.Data["gcscert.pem"]).To(Equal(encodedCert))
				g.Expect(s.Data["gcskey.pem"]).To(Equal(encodedKey))
				g.Expect(s.Data["extdata"]).To(Equal([]byte{'a'}))
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().Domain().AnyTimes().Return("example.com")
			_env.EXPECT().ClusterGenevaLoggingSecret().Return(validKey, validCerts[0])

			_, log := testlog.New()

			builder := fake.NewClientBuilder().WithRuntimeObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, builder.Build())
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterDocument(&api.OpenShiftClusterDocument{ID: clusterUUID, OpenShiftCluster: &api.OpenShiftCluster{Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						Domain: "something",
					},
				}}}),
			)

			err := EnsureMDSDCertificates(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.check != nil {
				tt.check(ch, g)
			}
		})
	}
}
