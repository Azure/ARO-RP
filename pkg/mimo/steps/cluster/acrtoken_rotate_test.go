package cluster

import (
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_armcontainerregistry "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcontainerregistry"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testacrtoken "github.com/Azure/ARO-RP/test/util/acrtoken"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestRotateACRToken(t *testing.T) {
	startOf2024 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expiredTime := startOf2024.AddDate(0, 0, -365)

	fpRGName := "firstpartyrg"
	fpRegistryName := "registryaro"

	acrID := "/subscriptions/00000000-1111-0000-0000-000000000000/resourcegroups/" + fpRGName + "/providers/Microsoft.ContainerRegistry/registries/" + fpRegistryName
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	tests := []struct {
		name     string // description of this test case
		azureEnv azureclient.AROEnvironment
		isDev    bool
		oc       func() api.OpenShiftClusterProperties
		objects  []client.Object

		fake   func(*testacrtoken.FakeACRToken)
		verify func(*require.Assertions, *testacrtoken.FakeACRToken) api.OpenShiftClusterProperties

		wantErr      error
		expectedLogs []testlog.ExpectedLogEntry
	}{
		{
			name:     "token is in expected validity duration, does not rotate",
			isDev:    false,
			azureEnv: azureclient.PublicCloud,
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
			name:     "token is expired",
			isDev:    false,
			azureEnv: azureclient.PublicCloud,
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
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "cluster",
						Namespace: "openshift-azure-operator",
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
				},
			},

			fake: func(*testacrtoken.FakeACRToken) {
			},
			verify: func(r *require.Assertions, t *testacrtoken.FakeACRToken) api.OpenShiftClusterProperties {
				generated := t.GetGeneratedPasswords()
				r.Len(generated, 1, "wrong number of passwords requested")

				return api.OpenShiftClusterProperties{
					RegistryProfiles: []*api.RegistryProfile{
						{
							Name:      publicACR,
							Username:  user,
							IssueDate: &expiredTime,
							Password:  api.SecureString(generated[0]),
						},
					},
				}
			},
			wantErr: nil,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("token has 0s validity remaining, should rotate in -5400h0m0s"),
				},
			},
		},
		{
			name:     "no registry profile",
			isDev:    false,
			azureEnv: azureclient.PublicCloud,
			oc: func() api.OpenShiftClusterProperties {
				return api.OpenShiftClusterProperties{}
			},
			wantErr: cluster.ErrNoRegistryProfileFound,
		},
		{
			name:     "does not run in test",
			isDev:    true,
			azureEnv: azureclient.PublicCloud,
			oc: func() api.OpenShiftClusterProperties {
				return api.OpenShiftClusterProperties{}
			},
			wantErr: cluster.ErrCannotRotateACRTokensInDev,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(tt.isDev)

			_env.EXPECT().IsCI().AnyTimes().Return(false)
			_env.EXPECT().ACRDomain().AnyTimes().Return(publicACR)
			_env.EXPECT().ACRResourceID().AnyTimes().Return(acrID)
			_env.EXPECT().Environment().AnyTimes().Return(&tt.azureEnv)
			_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
				return startOf2024
			})

			tokensClient := mock_armcontainerregistry.NewMockTokensClient(controller)
			registriesClient := mock_armcontainerregistry.NewMockRegistriesClient(controller)

			acrManager := testacrtoken.New()

			if tt.fake != nil {
				tt.fake(acrManager)
			}

			hook, log := testlog.LogForTesting(t)

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

			builder := fake.NewClientBuilder().WithObjects(tt.objects...)
			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				t.Context(), _env, log, func() time.Time { return startOf2024 },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterDocument(doc),
				testtasks.WithOpenShiftDatabase(openShiftClustersDatabase),
				testtasks.WithTokensClient(tokensClient),
				testtasks.WithRegistriesClient(registriesClient),
			)

			gotErr := rotateACRTokenWithManager(tc, acrManager, false)
			if tt.wantErr != nil {
				r.ErrorIs(gotErr, tt.wantErr)
				return
			} else {
				r.NoError(gotErr)
			}

			if tt.verify != nil {
				afterProps := tt.verify(r, acrManager)
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
			}

			err := testlog.AssertLoggingOutput(hook, tt.expectedLogs)
			r.NoError(err)
		})
	}
}
