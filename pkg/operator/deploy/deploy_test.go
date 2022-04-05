package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestCheckIngressIP(t *testing.T) {
	type test struct {
		name    string
		oc      func() *api.OpenShiftClusterProperties
		want    string
		wantErr error
	}

	for _, tt := range []*test{
		{
			name: "default IngressProfile",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "default",
							IP:   "1.2.3.4",
						},
					},
				}
			},
			want:    "1.2.3.4",
			wantErr: nil,
		},
		{
			name: "Multiple IngressProfiles, pick default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "custom-ingress",
							IP:   "1.1.1.1",
						},
						{
							Name: "default",
							IP:   "1.2.3.4",
						},
						{
							Name: "not-default",
							IP:   "1.1.2.2",
						},
					},
				}
			},
			want:    "1.2.3.4",
			wantErr: nil,
		},
		{
			name: "Single Ingress Profile, No Default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "custom-ingress",
							IP:   "1.1.1.1",
						},
					},
				}
			},
			want:    "1.1.1.1",
			wantErr: nil,
		},
		{
			name: "Multiple Ingress Profiles, No Default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{
						{
							Name: "custom-ingress",
							IP:   "1.1.1.1",
						},
						{
							Name: "not-default",
							IP:   "1.1.2.2",
						},
					},
				}
			},
			want:    "1.1.1.1",
			wantErr: nil,
		},
		{
			name: "No Ingresses in IngressProfiles, Error",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					IngressProfiles: []api.IngressProfile{},
				}
			},
			want:    "",
			wantErr: errors.New("No Ingress Profiles found"),
		},
		{
			name: "Nil IngressProfiles, Error",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{}
			},
			want:    "",
			wantErr: errors.New("No Ingress Profiles found"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := tt.oc()
			ingressIP, err := checkIngressIP(oc.IngressProfiles)
			if err != nil && err.Error() != tt.wantErr.Error() ||
				err == nil && tt.wantErr != nil {
				t.Error(err)
			}
			if tt.want != ingressIP {
				t.Error(cmp.Diff(ingressIP, tt.want))
			}
		})
	}
}
