package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBuildDesiredCloudLBConfig(t *testing.T) {
	tests := []struct {
		name      string
		apiIntIP  string
		ingressIP string
		want      *cloudLoadBalancerConfig
	}{
		{
			name:      "both IPs provided",
			apiIntIP:  "10.0.0.1",
			ingressIP: "10.0.0.2",
			want: &cloudLoadBalancerConfig{
				DNSType: "ClusterHosted",
				ClusterHosted: &cloudLoadBalancerIPs{
					APIIntLoadBalancerIPs:  []string{"10.0.0.1"},
					APILoadBalancerIPs:     []string{"10.0.0.1"},
					IngressLoadBalancerIPs: []string{"10.0.0.2"},
				},
			},
		},
		{
			name:      "apiIntIP is used for both api and apiInt per TDR",
			apiIntIP:  "10.0.0.5",
			ingressIP: "10.0.0.6",
			want: &cloudLoadBalancerConfig{
				DNSType: "ClusterHosted",
				ClusterHosted: &cloudLoadBalancerIPs{
					APIIntLoadBalancerIPs:  []string{"10.0.0.5"},
					APILoadBalancerIPs:     []string{"10.0.0.5"},
					IngressLoadBalancerIPs: []string{"10.0.0.6"},
				},
			},
		},
		{
			name:      "empty ingress IP omits ingressLoadBalancerIPs",
			apiIntIP:  "10.0.0.1",
			ingressIP: "",
			want: &cloudLoadBalancerConfig{
				DNSType: "ClusterHosted",
				ClusterHosted: &cloudLoadBalancerIPs{
					APIIntLoadBalancerIPs: []string{"10.0.0.1"},
					APILoadBalancerIPs:    []string{"10.0.0.1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDesiredCloudLBConfig(tt.apiIntIP, tt.ingressIP)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildDesiredCloudLBConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetCurrentCloudLBConfig(t *testing.T) {
	tests := []struct {
		name  string
		infra *unstructured.Unstructured
		want  *cloudLoadBalancerConfig
	}{
		{
			name: "no cloudLoadBalancerConfig returns nil",
			infra: &unstructured.Unstructured{
				Object: map[string]any{
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "full config present",
			infra: &unstructured.Unstructured{
				Object: map[string]any{
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{
								"cloudLoadBalancerConfig": map[string]any{
									"dnsType": "ClusterHosted",
									"clusterHosted": map[string]any{
										"apiIntLoadBalancerIPs":  []any{"10.0.0.1"},
										"apiLoadBalancerIPs":     []any{"10.0.0.1"},
										"ingressLoadBalancerIPs": []any{"10.0.0.2"},
									},
								},
							},
						},
					},
				},
			},
			want: &cloudLoadBalancerConfig{
				DNSType: "ClusterHosted",
				ClusterHosted: &cloudLoadBalancerIPs{
					APIIntLoadBalancerIPs:  []string{"10.0.0.1"},
					APILoadBalancerIPs:     []string{"10.0.0.1"},
					IngressLoadBalancerIPs: []string{"10.0.0.2"},
				},
			},
		},
		{
			name: "empty status returns nil",
			infra: &unstructured.Unstructured{
				Object: map[string]any{},
			},
			want: nil,
		},
		{
			name: "config without ingress IPs",
			infra: &unstructured.Unstructured{
				Object: map[string]any{
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{
								"cloudLoadBalancerConfig": map[string]any{
									"dnsType": "ClusterHosted",
									"clusterHosted": map[string]any{
										"apiIntLoadBalancerIPs": []any{"10.0.0.1"},
										"apiLoadBalancerIPs":    []any{"10.0.0.1"},
									},
								},
							},
						},
					},
				},
			},
			want: &cloudLoadBalancerConfig{
				DNSType: "ClusterHosted",
				ClusterHosted: &cloudLoadBalancerIPs{
					APIIntLoadBalancerIPs:  []string{"10.0.0.1"},
					APILoadBalancerIPs:     []string{"10.0.0.1"},
					IngressLoadBalancerIPs: nil,
				},
			},
		},
		{
			name: "dnsType present but no clusterHosted section",
			infra: &unstructured.Unstructured{
				Object: map[string]any{
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{
								"cloudLoadBalancerConfig": map[string]any{
									"dnsType": "ClusterHosted",
								},
							},
						},
					},
				},
			},
			want: &cloudLoadBalancerConfig{
				DNSType: "ClusterHosted",
				ClusterHosted: &cloudLoadBalancerIPs{
					APIIntLoadBalancerIPs:  nil,
					APILoadBalancerIPs:     nil,
					IngressLoadBalancerIPs: nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCurrentCloudLBConfig(tt.infra)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCurrentCloudLBConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToInterfaceSlice(t *testing.T) {
	tests := []struct {
		name string
		s    []string
		want []any
	}{
		{
			name: "non-empty slice",
			s:    []string{"a", "b", "c"},
			want: []any{"a", "b", "c"},
		},
		{
			name: "single element",
			s:    []string{"10.0.0.1"},
			want: []any{"10.0.0.1"},
		},
		{
			name: "empty slice",
			s:    []string{},
			want: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toInterfaceSlice(tt.s)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toInterfaceSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReconcileInfrastructureCR tests the end-to-end reconciliation of the Infrastructure CR.
//
// Note: The fake client converts unstructured objects to their typed counterparts
// (configv1.Infrastructure) when stored, which drops the cloudLoadBalancerConfig
// field since the vendored openshift/api struct doesn't include it. This means
// the "no-op" case (config already up to date) always sees current=nil and patches.
// The comparison/no-op logic is tested separately in TestBuildDesiredCloudLBConfig
// and TestGetCurrentCloudLBConfig using direct function calls.
func TestReconcileInfrastructureCR(t *testing.T) {
	tests := []struct {
		name      string
		apiIntIP  string
		ingressIP string
		infraObj  *unstructured.Unstructured
		wantErr   string
	}{
		{
			name:      "Infrastructure CR not found returns error",
			apiIntIP:  "10.0.0.1",
			ingressIP: "10.0.0.2",
			infraObj:  nil,
			wantErr:   "failed to get Infrastructure CR",
		},
		{
			name:      "successfully patches Infrastructure CR when config not present",
			apiIntIP:  "10.0.0.1",
			ingressIP: "10.0.0.2",
			infraObj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "config.openshift.io/v1",
					"kind":       "Infrastructure",
					"metadata": map[string]any{
						"name": "cluster",
					},
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{},
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name:      "successfully patches Infrastructure CR with both IPs",
			apiIntIP:  "10.0.0.3",
			ingressIP: "10.0.0.4",
			infraObj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "config.openshift.io/v1",
					"kind":       "Infrastructure",
					"metadata": map[string]any{
						"name": "cluster",
					},
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{},
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name:      "successfully patches Infrastructure CR without ingress IP",
			apiIntIP:  "10.0.0.1",
			ingressIP: "",
			infraObj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "config.openshift.io/v1",
					"kind":       "Infrastructure",
					"metadata": map[string]any{
						"name": "cluster",
					},
					"status": map[string]any{
						"platformStatus": map[string]any{
							"azure": map[string]any{},
						},
					},
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := ctrlfake.NewClientBuilder()
			if tt.infraObj != nil {
				builder = builder.WithObjects(tt.infraObj)
			}
			c := builder.Build()
			log := logrus.NewEntry(logrus.StandardLogger())

			err := reconcileInfrastructureCR(context.Background(), c, log, tt.apiIntIP, tt.ingressIP)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
