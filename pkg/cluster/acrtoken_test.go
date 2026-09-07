package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testenvtest "github.com/Azure/ARO-RP/test/envtest"
	testacrtoken "github.com/Azure/ARO-RP/test/util/acrtoken"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestRotateACRToken(t *testing.T) {
	publicACR := "arosvc.azurecr.io"
	user := "testuser"

	startOf2024 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expiredTime := startOf2024.AddDate(0, 0, -365)

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
	_env.EXPECT().IsCI().AnyTimes().Return(false)
	_env.EXPECT().ACRDomain().AnyTimes().Return(publicACR)
	_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
	_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
		return startOf2024
	})
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	tests := []struct {
		name    string // description of this test case
		oc      func() api.OpenShiftClusterProperties
		objects []client.Object

		verify func(*require.Assertions, *testacrtoken.FakeACRToken) (api.OpenShiftClusterProperties, []runtime.Object)

		wantErr      error
		expectedLogs []testlog.ExpectedLogEntry
	}{
		{
			name: "token is in expected validity duration, does not rotate",
			oc: func() api.OpenShiftClusterProperties {
				return api.OpenShiftClusterProperties{
					RegistryProfiles: []*api.RegistryProfile{
						{
							Name:      publicACR,
							Username:  user,
							IssueDate: &startOf2024,
						},
					},
				}
			},
			wantErr: nil,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("token has 4320h0m0s validity remaining, should rotate in 3360h0m0s"),
				},
			},
		},
		{
			name: "token is expired, is rotated",
			oc: func() api.OpenShiftClusterProperties {
				return api.OpenShiftClusterProperties{
					RegistryProfiles: []*api.RegistryProfile{
						{
							Name:      publicACR,
							Username:  user,
							IssueDate: &expiredTime,
						},
					},
				}
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "openshift-azure-operator",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "openshift-config",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster",
						Namespace: "openshift-azure-operator",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						"somethingElse":     {},
						".dockerconfigjson": []byte(`{}`),
					},
				},
			},
			verify: func(r *require.Assertions, t *testacrtoken.FakeACRToken) (api.OpenShiftClusterProperties, []runtime.Object) {
				generated := t.GetGeneratedPasswords()
				r.Len(generated, 1, "wrong number of passwords requested")

				props := api.OpenShiftClusterProperties{
					RegistryProfiles: []*api.RegistryProfile{
						{
							Name:      publicACR,
							Username:  user,
							IssueDate: &startOf2024,
							Password:  api.SecureString(generated[0]),
						},
					},
				}

				// The two secrets are laid down with the correct content
				b64pwpair := base64.StdEncoding.EncodeToString([]byte(user + ":" + generated[0]))
				objs := []runtime.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pull-secret",
							Namespace: "openshift-config",
						},
						Type: corev1.SecretTypeDockerConfigJson,
						Data: map[string][]byte{
							"somethingElse":     {},
							".dockerconfigjson": []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"` + b64pwpair + `"}}}`),
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "openshift-azure-operator",
						},
						Type: corev1.SecretTypeOpaque,
						Data: map[string][]byte{
							".dockerconfigjson": []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"` + b64pwpair + `"}}}`),
						},
					},
				}

				return props, objs
			},
			wantErr: nil,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("token has 0s validity remaining, should rotate in -5400h0m0s"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("rotating ACR token"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Apply v1/Secret/openshift-azure-operator/cluster"),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("Apply v1/Secret/openshift-config/pull-secret"),
				},
			},
		},
		{
			name: "no registry profile",
			oc: func() api.OpenShiftClusterProperties {
				return api.OpenShiftClusterProperties{}
			},
			wantErr: ErrNoRegistryProfileFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testenvtest.SkipIfNotUsingEnvtest(t)
			r := require.New(t)

			doc := &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:         key,
					Location:   "eastus",
					Properties: tt.oc(),
				},
			}

			openShiftClustersDatabase, openShiftClustersClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(doc)
			fixture.Create()

			acrManager := testacrtoken.New(func() time.Time {
				return startOf2024
			})
			hook, log := testlog.LogForTesting(t)

			restConfig := testenvtest.StartTestEnvironment(t)
			cl, err := client.New(restConfig, client.Options{})
			r.NoError(err)

			ch := clienthelper.NewWithClient(log, cl)

			for _, i := range tt.objects {
				err = ch.Create(t.Context(), i)
				r.NoError(err)
			}

			m := manager{
				log: log,
				env: _env,
				ch:  ch,

				doc: doc,

				db: openShiftClustersDatabase,

				newACRTokenManager: func(i env.Interface) (acrtoken.Manager, error) { return acrManager, nil },
			}

			gotErr := m.rotateACRTokenPassword(t.Context())
			if tt.wantErr != nil {
				r.ErrorIs(gotErr, tt.wantErr)
				return
			} else {
				r.NoError(gotErr)
			}

			if tt.verify != nil {
				afterProps, afterObjects := tt.verify(r, acrManager)
				afterDoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:         key,
						Location:   "eastus",
						Properties: afterProps,
					},
				}
				checker := testdatabase.NewChecker()
				checker.AddOpenShiftClusterDocuments(afterDoc)
				r.Empty(checker.CheckOpenShiftClusters(openShiftClustersClient))

				if afterObjects != nil {
					objs := []runtime.Object{}
					for _, i := range afterObjects {
						f := reflect.New(reflect.TypeOf(i).Elem()).Interface().(runtime.Object)
						err := ch.GetOne(t.Context(), client.ObjectKeyFromObject(i.(client.Object)), f)
						r.NoError(err)
						objs = append(objs, f)
					}
					testclienthelper.CompareObjectList(t, objs, afterObjects)
				}
			}

			err = testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			r.NoError(err)
		})
	}
}
