package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdkdns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armdns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armdns"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()

	managedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_armdns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed, new record",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{},
					}, autorest.DetailedError{
						StatusCode: http.StatusNotFound,
					})

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, sdkdns.RecordSet{
						Properties: &sdkdns.RecordSetProperties{
							Metadata: map[string]*string{
								resourceID: to.StringPtr(tt.oc.ID),
							},
							TTL: to.Int64Ptr(300),
						},
					}, &sdkdns.RecordSetsClientCreateOrUpdateOptions{
						IfMatch:     to.StringPtr(""),
						IfNoneMatch: to.StringPtr("*"),
					}).
					Return(sdkdns.RecordSetsClientCreateOrUpdateResponse{
						RecordSet: sdkdns.RecordSet{},
					}, nil)
			},
		},
		{
			name: "managed, our record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Properties: &sdkdns.RecordSetProperties{
								Metadata: map[string]*string{
									"resourceId": &tt.oc.ID,
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "managed, someone else's record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Properties: &sdkdns.RecordSetProperties{
								Metadata: map[string]*string{
									"resourceId": to.StringPtr("not us"),
								},
							},
						},
					}, nil)
			},
			wantErr: `400: DuplicateDomain: : The provided domain 'domain' is already in use by a cluster.`,
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{}, fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
		{
			name: "unmanaged",
			oc:   unmanagedOc,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")
			env.EXPECT().Domain().AnyTimes().Return("domain")

			recordsets := mock_armdns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.Create(ctx, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	managedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "test.domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_armdns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed, our record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.test", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Etag: to.StringPtr("etag"),
							Properties: &sdkdns.RecordSetProperties{
								Metadata: map[string]*string{
									"resourceId": &tt.oc.ID,
								},
							},
						}}, nil)

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "api.test", sdkdns.RecordTypeA, sdkdns.RecordSet{
						Properties: &sdkdns.RecordSetProperties{
							Metadata: map[string]*string{
								resourceID: to.StringPtr(tt.oc.ID),
							},
							TTL: to.Int64Ptr(300),
							ARecords: []*sdkdns.ARecord{
								{
									IPv4Address: to.StringPtr("1.2.3.4"),
								},
							},
						},
					}, &sdkdns.RecordSetsClientCreateOrUpdateOptions{
						IfMatch:     to.StringPtr("etag"),
						IfNoneMatch: to.StringPtr(""),
					}).
					Return(sdkdns.RecordSetsClientCreateOrUpdateResponse{
						RecordSet: sdkdns.RecordSet{},
					}, nil)
			},
		},
		{
			name: "managed, someone else's record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.test", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Properties: &sdkdns.RecordSetProperties{
								Metadata: map[string]*string{
									"resourceId": to.StringPtr("not us"),
								},
							},
						}}, nil)
			},
			wantErr: `recordset "api.test" already registered`,
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.test", sdkdns.RecordTypeA, nil).
					Return(
						sdkdns.RecordSetsClientGetResponse{
							RecordSet: sdkdns.RecordSet{},
						}, fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
		{
			name: "unmanaged",
			oc:   unmanagedOc,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")
			env.EXPECT().Domain().AnyTimes().Return("domain")

			recordsets := mock_armdns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.Update(ctx, tt.oc, "1.2.3.4")
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestCreateOrUpdateRouter(t *testing.T) {
	ctx := context.Background()

	managedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name     string
		routerIP string
		oc       *api.OpenShiftCluster
		mocks    func(*test, *mock_armdns.MockRecordSetsClient)
		wantErr  string
	}

	for _, tt := range []*test{
		{
			name:     "managed - create",
			routerIP: "1.2.3.4",
			oc:       managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{},
					}, fmt.Errorf("random error"))

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, sdkdns.RecordSet{
						Properties: &sdkdns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: []*sdkdns.ARecord{
								{
									IPv4Address: to.StringPtr(tt.routerIP),
								},
							},
						},
					}, &sdkdns.RecordSetsClientCreateOrUpdateOptions{
						IfMatch:     nil,
						IfNoneMatch: nil,
					}).
					Return(sdkdns.RecordSetsClientCreateOrUpdateResponse{
						RecordSet: sdkdns.RecordSet{},
					}, nil)
			},
		},
		{
			name:     "managed, error",
			routerIP: "1.2.3.4",
			oc:       managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{},
					}, fmt.Errorf("random error"))

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, sdkdns.RecordSet{
						Properties: &sdkdns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: []*sdkdns.ARecord{
								{
									IPv4Address: to.StringPtr(tt.routerIP),
								},
							},
						},
					}, &sdkdns.RecordSetsClientCreateOrUpdateOptions{
						IfMatch:     nil,
						IfNoneMatch: nil,
					}).
					Return(sdkdns.RecordSetsClientCreateOrUpdateResponse{
						RecordSet: sdkdns.RecordSet{},
					}, fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
		{
			name:     "managed, update match",
			routerIP: "1.2.3.4",
			oc:       managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Properties: &sdkdns.RecordSetProperties{
								TTL: to.Int64Ptr(300),
								ARecords: []*sdkdns.ARecord{
									{
										IPv4Address: to.StringPtr(tt.routerIP),
									},
								},
							},
						},
					}, nil)
			},
		},
		{
			name:     "managed, update missmatch",
			routerIP: "2.2.3.4",
			oc:       managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Properties: &sdkdns.RecordSetProperties{
								TTL: to.Int64Ptr(300),
								ARecords: []*sdkdns.ARecord{
									{
										IPv4Address: to.StringPtr("1.2.3.4"),
									},
								},
							},
						},
					}, nil)

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, sdkdns.RecordSet{
						Properties: &sdkdns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: []*sdkdns.ARecord{
								{
									IPv4Address: to.StringPtr(tt.routerIP),
								},
							},
						},
					}, &sdkdns.RecordSetsClientCreateOrUpdateOptions{
						IfMatch:     nil,
						IfNoneMatch: nil,
					}).
					Return(sdkdns.RecordSetsClientCreateOrUpdateResponse{
						RecordSet: sdkdns.RecordSet{},
					}, nil)
			},
		},
		{
			name: "unmanaged",
			oc:   unmanagedOc,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")
			env.EXPECT().Domain().AnyTimes().Return("domain")

			recordsets := mock_armdns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.CreateOrUpdateRouter(ctx, tt.oc, tt.routerIP)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	managedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_armdns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed, not found",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{},
					}, autorest.DetailedError{
						StatusCode: http.StatusNotFound,
					})
			},
		},
		{
			name: "managed, our record exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Etag: to.StringPtr("etag"),
							Properties: &sdkdns.RecordSetProperties{
								Metadata: map[string]*string{
									"resourceId": &tt.oc.ID,
								},
							},
						},
					}, nil)

				recordsets.EXPECT().
					Delete(ctx, "rpResourcegroup", "domain", "*.apps.domain", sdkdns.RecordTypeA, &sdkdns.RecordSetsClientDeleteOptions{
						IfMatch: to.StringPtr(""),
					}).
					Return(sdkdns.RecordSetsClientDeleteResponse{}, nil)

				recordsets.EXPECT().
					Delete(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, &sdkdns.RecordSetsClientDeleteOptions{
						IfMatch: to.StringPtr("etag"),
					}).
					Return(sdkdns.RecordSetsClientDeleteResponse{}, nil)
			},
		},
		{
			name: "managed, someone else's record exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{
							Properties: &sdkdns.RecordSetProperties{
								Metadata: map[string]*string{
									"resourceId": to.StringPtr("not us"),
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_armdns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", sdkdns.RecordTypeA, nil).
					Return(sdkdns.RecordSetsClientGetResponse{
						RecordSet: sdkdns.RecordSet{},
					}, fmt.Errorf("random error"))
			},
			wantErr: "random error",
		},
		{
			name: "unmanaged",
			oc:   unmanagedOc,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")
			env.EXPECT().Domain().AnyTimes().Return("domain")

			recordsets := mock_armdns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.Delete(ctx, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestManagedDomain(t *testing.T) {
	for _, tt := range []struct {
		domain  string
		want    string
		wantErr string
	}{
		{
			domain: "eastus.aroapp.io",
		},
		{
			domain: "aroapp.io",
		},
		{
			domain: "redhat.com",
		},
		{
			domain: "foo.eastus.aroapp.io.redhat.com",
		},
		{
			domain: "foo.eastus.aroapp.io",
			want:   "foo.eastus.aroapp.io",
		},
		{
			domain: "bar",
			want:   "bar.eastus.aroapp.io",
		},
		{
			domain:  "",
			wantErr: `invalid domain ""`,
		},
		{
			domain:  ".foo",
			wantErr: `invalid domain ".foo"`,
		},
		{
			domain:  "foo.",
			wantErr: `invalid domain "foo."`,
		},
	} {
		t.Run(tt.domain, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Domain().AnyTimes().Return("eastus.aroapp.io")

			got, err := ManagedDomain(env, tt.domain)
			if got != tt.want {
				t.Error(got)
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestManagedDomainPrefix(t *testing.T) {
	for _, tt := range []struct {
		domain  string
		want    string
		wantErr string
	}{
		{
			domain: "foo",
			want:   "foo",
		},
		{
			domain: "foo.domain",
			want:   "foo",
		},
		{
			domain: "foo.other",
			want:   "",
		},
		{
			domain:  "",
			wantErr: `invalid domain ""`,
		},
		{
			domain:  ".foo",
			wantErr: `invalid domain ".foo"`,
		},
		{
			domain:  "foo.",
			wantErr: `invalid domain "foo."`,
		},
	} {
		t.Run(tt.domain, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ResourceGroup().AnyTimes().Return("rpResourcegroup")
			env.EXPECT().Domain().AnyTimes().Return("domain")

			m := &manager{
				env: env,
			}

			got, err := m.managedDomainPrefix(tt.domain)
			if got != tt.want {
				t.Error(got)
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
