package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

type dummyOpenShiftClusterValidator struct{}

func (*dummyOpenShiftClusterValidator) Static(interface{}, *api.OpenShiftCluster) error {
	return nil
}

func (*dummyOpenShiftClusterValidator) Dynamic(context.Context, *api.OpenShiftCluster) error {
	return nil
}

func expectAsyncOperationDocumentCreate(asyncOperations *mock_database.MockAsyncOperations, key string, provisioningState api.ProvisioningState) {
	asyncOperations.EXPECT().
		Create(gomock.Any(), (*matcher.AsyncOperationDocument)(
			&api.AsyncOperationDocument{
				OpenShiftClusterKey: key,
				AsyncOperation: &api.AsyncOperation{
					InitialProvisioningState: provisioningState,
					ProvisioningState:        provisioningState,
				},
			}),
		)
}

func TestPutOrPatchOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"2019-12-31-preview": {
			OpenShiftClusterConverter: api.APIs["2019-12-31-preview"].OpenShiftClusterConverter,
			OpenShiftClusterValidator: func(env.Interface, string) api.OpenShiftClusterValidator {
				return &dummyOpenShiftClusterValidator{}
			},
			OpenShiftClusterCredentialsConverter: api.APIs["2019-12-31-preview"].OpenShiftClusterCredentialsConverter,
		},
	}

	clientkey, clientcerts, err := utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{clientcerts[0].Raw},
						PrivateKey:  clientkey,
					},
				},
			},
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		resourceID     string
		request        func(*v20191231preview.OpenShiftCluster)
		isPatch        bool
		mocks          func(*test, *mock_database.MockAsyncOperations, *mock_database.MockOpenShiftClusters)
		wantStatusCode int
		wantResponse   func(*test) *v20191231preview.OpenShiftCluster
		wantAsync      bool
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "create a new cluster",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateCreating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key:    strings.ToLower(tt.resourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.Properties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version: "4.3.0",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								TenantID: "11111111-1111-1111-1111-111111111111",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Create(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusCreated,
			wantResponse: func(tt *test) *v20191231preview.OpenShiftCluster {
				return &v20191231preview.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: v20191231preview.Properties{
						ProvisioningState: v20191231preview.ProvisioningStateCreating,
						ClusterProfile: v20191231preview.ClusterProfile{
							Version: "4.3.0",
						},
					},
				}
			},
		},
		{
			name:       "update a cluster from succeeded",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Tags: map[string]string{"tag": "will-be-removed"},
							Properties: api.Properties{
								ProvisioningState: api.ProvisioningStateSucceeded,
								IngressProfiles:   []api.IngressProfile{{Name: "will-be-removed"}},
								WorkerProfiles:    []api.WorkerProfile{{Name: "will-be-removed"}},
							},
						},
					}, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.Properties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20191231preview.OpenShiftCluster {
				return &v20191231preview.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: v20191231preview.Properties{
						ProvisioningState: v20191231preview.ProvisioningStateUpdating,
						ClusterProfile: v20191231preview.ClusterProfile{
							Domain: "changed",
						},
					},
				}
			},
		},
		{
			name:       "update a cluster from failed during update",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Tags: map[string]string{"tag": "will-be-removed"},
							Properties: api.Properties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateUpdating,
								IngressProfiles:         []api.IngressProfile{{Name: "will-be-removed"}},
								WorkerProfiles:          []api.WorkerProfile{{Name: "will-be-removed"}},
							},
						},
					}, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.Properties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20191231preview.OpenShiftCluster {
				return &v20191231preview.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: v20191231preview.Properties{
						ProvisioningState: v20191231preview.ProvisioningStateUpdating,
						ClusterProfile: v20191231preview.ClusterProfile{
							Domain: "changed",
						},
					},
				}
			},
		},
		{
			name:       "update a cluster from failed during creation",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.Properties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateCreating,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name:       "update a cluster from failed during deletion",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.Properties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateDeleting,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name:       "patch a cluster from succeeded",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
				oc.Properties.IngressProfiles = []v20191231preview.IngressProfile{{Name: "changed"}}
				oc.Properties.WorkerProfiles = []v20191231preview.WorkerProfile{{Name: "changed"}}
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Tags: map[string]string{"tag": "will-be-kept"},
							Properties: api.Properties{
								ProvisioningState: api.ProvisioningStateSucceeded,
								IngressProfiles:   []api.IngressProfile{{Name: "default"}},
								WorkerProfiles:    []api.WorkerProfile{{Name: "default"}},
							},
						},
					}, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.Properties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
							IngressProfiles: []api.IngressProfile{{Name: "changed"}},
							WorkerProfiles:  []api.WorkerProfile{{Name: "changed"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20191231preview.OpenShiftCluster {
				return &v20191231preview.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: v20191231preview.Properties{
						ProvisioningState: v20191231preview.ProvisioningStateUpdating,
						ClusterProfile: v20191231preview.ClusterProfile{
							Domain: "changed",
						},
						IngressProfiles: []v20191231preview.IngressProfile{{Name: "changed"}},
						WorkerProfiles:  []v20191231preview.WorkerProfile{{Name: "changed"}},
					},
				}
			},
		},
		{
			name:       "patch a cluster from failed during update",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Tags: map[string]string{"tag": "will-be-kept"},
							Properties: api.Properties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateUpdating,
								IngressProfiles:         []api.IngressProfile{{Name: "will-be-kept"}},
								WorkerProfiles:          []api.WorkerProfile{{Name: "will-be-kept"}},
							},
						},
					}, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.Properties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
							IngressProfiles: []api.IngressProfile{{Name: "will-be-kept"}},
							WorkerProfiles:  []api.WorkerProfile{{Name: "will-be-kept"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20191231preview.OpenShiftCluster {
				return &v20191231preview.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: v20191231preview.Properties{
						ProvisioningState: v20191231preview.ProvisioningStateUpdating,
						ClusterProfile: v20191231preview.ClusterProfile{
							Domain: "changed",
						},
						IngressProfiles: []v20191231preview.IngressProfile{{Name: "will-be-kept"}},
						WorkerProfiles:  []v20191231preview.WorkerProfile{{Name: "will-be-kept"}},
					},
				}
			},
		},
		{
			name:       "patch a cluster from failed during creation",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.Properties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateCreating,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name:       "patch a cluster from failed during deletion",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			request: func(oc *v20191231preview.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.Properties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateDeleting,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer cli.CloseIdleConnections()

			l := listener.NewListener()
			defer l.Close()

			env := &env.Test{
				L:        l,
				TLSKey:   serverkey,
				TLSCerts: servercerts,
			}
			env.SetClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))

			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			asyncOperations := mock_database.NewMockAsyncOperations(controller)
			openShiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			subscriptions := mock_database.NewMockSubscriptions(controller)

			tt.mocks(tt, asyncOperations, openShiftClusters)

			subscriptions.EXPECT().
				Get(gomock.Any(), mockSubID).
				Return(&api.SubscriptionDocument{
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				}, nil)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				AsyncOperations:   asyncOperations,
				OpenShiftClusters: openShiftClusters,
				Subscriptions:     subscriptions,
			}, apis, &noop.Noop{})
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).bucketAllocator = bucket.Fixed(1)

			go f.Run(ctx, nil, nil)

			buf := &bytes.Buffer{}
			oc := &v20191231preview.OpenShiftCluster{}
			if tt.request != nil {
				tt.request(oc)
			}
			err = json.NewEncoder(buf).Encode(oc)
			if err != nil {
				t.Fatal(err)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}
			req, err := http.NewRequest(method, "https://server"+tt.resourceID+"?api-version=2019-12-31-preview", buf)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = http.Header{
				"Content-Type": []string{"application/json"},
			}
			resp, err := cli.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantError == "" {
				var oc *v20191231preview.OpenShiftCluster
				err = json.Unmarshal(b, &oc)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(oc, tt.wantResponse(tt)) {
					b, _ := json.Marshal(oc)
					t.Error(string(b))
				}

			} else {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}

				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
			}
		})
	}
}
