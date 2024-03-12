package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/operator/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestCheck(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name            string
		DNS             *operatorv1.DNS
		wantErr         string
		wantMetricValid bool
	}{
		{
			name: "valid dns config",
			DNS: &operatorv1.DNS{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			wantMetricValid: true,
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
			wantErr:         `malformed config: "." in zones`,
			wantMetricValid: false,
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
			wantErr:         `custom upstream DNS servers in use: first-fake.io, second-fake.io, third-fake.io`,
			wantMetricValid: false,
		},
		{
			name:            "default config not found",
			wantErr:         `dnses.operator.openshift.io "default" not found`,
			wantMetricValid: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.DNS != nil {
				clientBuilder = clientBuilder.WithObjects(tt.DNS)
			}

			controller := gomock.NewController(t)
			defer controller.Finish()

			metricsClientFake := mock_metrics.NewMockClient(controller)
			metricsClientFake.EXPECT().UpdateDnsConfigurationValid(tt.wantMetricValid)

			sp := &checker{
				client:        clientBuilder.Build(),
				metricsClient: metricsClientFake,
			}

			err := sp.Check(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
