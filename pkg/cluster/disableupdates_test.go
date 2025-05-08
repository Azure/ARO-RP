package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestDisableUpdates(t *testing.T) {
	versionName := "version"
	acrDomain := "arotestsvc.azurecr.io"

	for _, tt := range []struct {
		name           string
		versioner      *FakeOpenShiftClusterDocumentVersionerService
		installViaHive bool

		wantVersion string
		wantImage   string
	}{
		{
			name: "Installing without hive - performs all expected modifications",
		},
		{
			name: "Installing with hive - performs all expected modifications",
			versioner: &FakeOpenShiftClusterDocumentVersionerService{
				expectedOpenShiftVersion: &api.OpenShiftVersion{
					Properties: api.OpenShiftVersionProperties{
						Version:           "4.0.0",
						OpenShiftPullspec: "arotestsvc.azurecr.io/openshift-release-dev/ocp-release@sha256:0000000000000000000000000000000000000000000000000000000000000000",
					},
				},
			},
			installViaHive: true,

			wantVersion: "4.0.0",
			wantImage:   "quay.io/openshift-release-dev/ocp-release@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name: "Installing with hive - removes metadata from version string if present",
			versioner: &FakeOpenShiftClusterDocumentVersionerService{
				expectedOpenShiftVersion: &api.OpenShiftVersion{
					Properties: api.OpenShiftVersionProperties{
						Version:           "4.0.0+installerref-abcdef",
						OpenShiftPullspec: "arotestsvc.azurecr.io/openshift-release-dev/ocp-release@sha256:0000000000000000000000000000000000000000000000000000000000000000",
					},
				},
			},
			installViaHive: true,

			wantVersion: "4.0.0",
			wantImage:   "quay.io/openshift-release-dev/ocp-release@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().ACRDomain().AnyTimes().Return(acrDomain)

			m := &manager{
				configcli: configfake.NewSimpleClientset(&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: versionName,
					},
					Spec: configv1.ClusterVersionSpec{
						Upstream: "RemoveMe",
						Channel:  "RemoveMe",
					},
				}),
				openShiftClusterDocumentVersioner: tt.versioner,
				installViaHive:                    tt.installViaHive,
				env:                               env,
			}

			err := m.disableUpdates(ctx)
			if err != nil {
				t.Error(err)
			}

			cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, versionName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if cv.Spec.Upstream != "" {
				t.Errorf("wanted no upstream but got %s", cv.Spec.Upstream)
			}
			if cv.Spec.Channel != "" {
				t.Errorf("wanted no channel but got %s", cv.Spec.Channel)
			}

			if tt.wantVersion != "" && cv.Spec.DesiredUpdate.Version != tt.wantVersion {
				t.Errorf("wanted version %s but got %s", tt.wantVersion, cv.Spec.DesiredUpdate.Version)
			}
			if tt.wantImage != "" && cv.Spec.DesiredUpdate.Image != tt.wantImage {
				t.Errorf("wanted image %s but got %s", tt.wantImage, cv.Spec.DesiredUpdate.Image)
			}
		})
	}
}
