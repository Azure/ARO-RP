package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "github.com/openshift/api/operator/v1"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestCheck(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name       string
		DNS        *operatorv1.DNS
		wantErr    string
		wantResult result
	}{
		{
			name: "valid dns config",
			DNS: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			wantResult: result{
				success: true,
				message: "no in-cluster upstream DNS servers",
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
			wantResult: result{
				success: false,
				message: `malformed config: "." in zones`,
			},
		},
		{
			name: "forward plugin upstream is",
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
			wantResult: result{
				success: true,
				message: `custom upstream DNS servers in use: first-fake.io, second-fake.io, third-fake.io`,
			},
		},
		{
			name:    "default config not found",
			wantErr: `dnses.operator.openshift.io "default" not found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.DNS != nil {
				clientBuilder = clientBuilder.WithObjects(tt.DNS)
			}

			sp := &checker{
				client: clientBuilder.Build(),
			}

			result, err := sp.Check(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
