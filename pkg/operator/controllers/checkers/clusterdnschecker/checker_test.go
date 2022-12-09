package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheck(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		DNS     *operatorv1.DNS
		wantErr string
	}{
		{
			name: "valid dns config",
			DNS: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
		},
		{
			name: "invalid config: malformed dns config",
			DNS: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.DNSSpec{
					Servers: []operatorv1.Server{
						{
							Zones: []string{"."},
						},
					},
				},
			},
			wantErr: `malformed config: "." in zones`,
		},
		{
			name: "invalid config: forward plugin upstream is",
			DNS: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Spec: operatorv1.DNSSpec{
					Servers: []operatorv1.Server{
						{
							ForwardPlugin: operatorv1.ForwardPlugin{
								Upstreams: []string{"first-fake.io", "second-fake.io"},
							},
						},
						{
							ForwardPlugin: operatorv1.ForwardPlugin{
								Upstreams: []string{"third-fake.io"},
							},
						},
					},
				},
			},
			wantErr: `custom upstream DNS servers in use: first-fake.io, second-fake.io, third-fake.io`,
		},
		{
			name:    "default config not found",
			wantErr: `dnses.operator.openshift.io "default" not found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			operatorcliMock := operatorfake.NewSimpleClientset()
			if tt.DNS != nil {
				operatorcliMock.Tracker().Add(tt.DNS)
			}

			sp := &checker{
				operatorcli: operatorcliMock,
			}

			err := sp.Check(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%s\n !=\n%s", err, tt.wantErr)
			}
		})
	}
}
