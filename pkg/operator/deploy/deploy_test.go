package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
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
			wantErr: errors.New("no Ingress Profiles found"),
		},
		{
			name: "Nil IngressProfiles, Error",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{}
			},
			want:    "",
			wantErr: errors.New("no Ingress Profiles found"),
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

func TestOperatorVersion(t *testing.T) {
	type test struct {
		name         string
		oc           func() *api.OpenShiftClusterProperties
		wantVersion  string
		wantPullspec string
	}

	for _, tt := range []*test{
		{
			name: "default",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{}
			},
			wantVersion:  version.GitCommit,
			wantPullspec: "defaultaroimagefromenv",
		},
		{
			name: "overridden",
			oc: func() *api.OpenShiftClusterProperties {
				return &api.OpenShiftClusterProperties{
					OperatorVersion: "v20220101.0",
				}
			},
			wantVersion:  "v20220101.0",
			wantPullspec: "intsvcdomain/aro:v20220101.0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := tt.oc()

			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRDomain().AnyTimes().Return("intsvcdomain")
			_env.EXPECT().AROOperatorImage().AnyTimes().Return("defaultaroimagefromenv")
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

			o := &operator{
				oc:  &api.OpenShiftCluster{Properties: *oc},
				env: _env,
			}

			staticResources, err := o.createObjects()
			if err != nil {
				t.Error(err)
			}

			var deployments []*appsv1.Deployment
			for _, i := range staticResources {
				if d, ok := i.(*appsv1.Deployment); ok {
					deployments = append(deployments, d)
				}
			}

			if len(deployments) != 2 {
				t.Errorf("found %d deployments, not 2", len(deployments))
			}

			for _, d := range deployments {
				if d.Labels["version"] != tt.wantVersion {
					t.Errorf("Got %q, not %q for label \"version\"", d.Labels["version"], tt.wantVersion)
				}

				if len(d.Spec.Template.Spec.Containers) != 1 {
					t.Errorf("found %d containers, not 1", len(d.Spec.Template.Spec.Containers))
				}

				image := d.Spec.Template.Spec.Containers[0].Image
				if image != tt.wantPullspec {
					t.Errorf("Got %q, not %q for the image", image, tt.wantPullspec)
				}
			}
		})
	}
}
