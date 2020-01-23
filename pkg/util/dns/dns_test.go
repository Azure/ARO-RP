package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	mock_dns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/dns"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()

	env := &env.Test{
		TestResourceGroup: "rpResourcegroup",
		TestDomain:        "domain",
	}

	managedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_dns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed, new record",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{}, autorest.DetailedError{
						StatusCode: http.StatusNotFound,
					})

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A, mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								resourceID: to.StringPtr(tt.oc.ID),
							},
							TTL: to.Int64Ptr(300),
						},
					}, "", "*").
					Return(mgmtdns.RecordSet{}, nil)
			},
		},
		{
			name: "managed, our record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								"resourceId": &tt.oc.ID,
							},
						},
					}, nil)
			},
		},
		{
			name: "managed, someone else's record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								"resourceId": to.StringPtr("not us"),
							},
						},
					}, nil)
			},
			wantErr: `recordset "api.domain" already registered`,
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{}, fmt.Errorf("random error"))
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

			recordsets := mock_dns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.Create(ctx, tt.oc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	env := &env.Test{
		TestResourceGroup: "rpResourcegroup",
		TestDomain:        "domain",
	}

	managedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "test.domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_dns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed, our record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.test", mgmtdns.A).
					Return(mgmtdns.RecordSet{
						Etag: to.StringPtr("etag"),
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								"resourceId": &tt.oc.ID,
							},
						},
					}, nil)

				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "api.test", mgmtdns.A, mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								resourceID: to.StringPtr(tt.oc.ID),
							},
							TTL: to.Int64Ptr(300),
							ARecords: &[]mgmtdns.ARecord{
								{
									Ipv4Address: to.StringPtr("1.2.3.4"),
								},
							},
						},
					}, "etag", "").
					Return(mgmtdns.RecordSet{}, nil)
			},
		},
		{
			name: "managed, someone else's record already exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.test", mgmtdns.A).
					Return(mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								"resourceId": to.StringPtr("not us"),
							},
						},
					}, nil)
			},
			wantErr: `recordset "api.test" already registered`,
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.test", mgmtdns.A).
					Return(mgmtdns.RecordSet{}, fmt.Errorf("random error"))
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

			recordsets := mock_dns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.Update(ctx, tt.oc, "1.2.3.4")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestCreateOrUpdateRouter(t *testing.T) {
	ctx := context.Background()

	env := &env.Test{
		TestResourceGroup: "rpResourcegroup",
		TestDomain:        "domain",
	}

	managedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_dns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "*.apps.domain", mgmtdns.A, mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: &[]mgmtdns.ARecord{
								{
									Ipv4Address: to.StringPtr("1.2.3.4"),
								},
							},
						},
					}, "", "").
					Return(mgmtdns.RecordSet{}, nil)
			},
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					CreateOrUpdate(ctx, "rpResourcegroup", "domain", "*.apps.domain", mgmtdns.A, mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							TTL: to.Int64Ptr(300),
							ARecords: &[]mgmtdns.ARecord{
								{
									Ipv4Address: to.StringPtr("1.2.3.4"),
								},
							},
						},
					}, "", "").
					Return(mgmtdns.RecordSet{}, fmt.Errorf("random error"))
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

			recordsets := mock_dns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.CreateOrUpdateRouter(ctx, tt.oc, "1.2.3.4")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	env := &env.Test{
		TestResourceGroup: "rpResourcegroup",
		TestDomain:        "domain",
	}

	managedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain",
			},
		},
	}

	unmanagedOc := &api.OpenShiftCluster{
		Properties: api.Properties{
			ClusterProfile: api.ClusterProfile{
				Domain: "domain.notmanaged",
			},
		},
	}

	type test struct {
		name    string
		oc      *api.OpenShiftCluster
		mocks   func(*test, *mock_dns.MockRecordSetsClient)
		wantErr string
	}

	for _, tt := range []*test{
		{
			name: "managed, not found",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{}, autorest.DetailedError{
						StatusCode: http.StatusNotFound,
					})
			},
		},
		{
			name: "managed, our record exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{
						Etag: to.StringPtr("etag"),
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								"resourceId": &tt.oc.ID,
							},
						},
					}, nil)

				recordsets.EXPECT().
					Delete(ctx, "rpResourcegroup", "domain", "*.apps.domain", mgmtdns.A, "").
					Return(autorest.Response{}, nil)

				recordsets.EXPECT().
					Delete(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A, "etag").
					Return(autorest.Response{}, nil)
			},
		},
		{
			name: "managed, someone else's record exists",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{
						RecordSetProperties: &mgmtdns.RecordSetProperties{
							Metadata: map[string]*string{
								"resourceId": to.StringPtr("not us"),
							},
						},
					}, nil)
			},
		},
		{
			name: "managed, error",
			oc:   managedOc,
			mocks: func(tt *test, recordsets *mock_dns.MockRecordSetsClient) {
				recordsets.EXPECT().
					Get(ctx, "rpResourcegroup", "domain", "api.domain", mgmtdns.A).
					Return(mgmtdns.RecordSet{}, fmt.Errorf("random error"))
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

			recordsets := mock_dns.NewMockRecordSetsClient(controller)
			if tt.mocks != nil {
				tt.mocks(tt, recordsets)
			}

			m := &manager{
				env:        env,
				recordsets: recordsets,
			}

			err := m.Delete(ctx, tt.oc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestManagedDomainPrefix(t *testing.T) {
	m := &manager{
		env: &env.Test{
			TestResourceGroup: "rpResourcegroup",
			TestDomain:        "domain",
		},
	}

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
			got, err := m.managedDomainPrefix(tt.domain)
			if got != tt.want {
				t.Error(got)
			}
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
