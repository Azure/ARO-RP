package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestUpgradeCluster(t *testing.T) {
	ctx := context.Background()

	Stream43 := version.Stream{
		Version: version.NewVersion(4, 3, 18),
	}
	Stream44 := version.Stream{
		Version: version.NewVersion(4, 4, 3),
	}

	version.Streams = append([]version.Stream{}, Stream43, Stream44)

	newFakecli := func(channel, version string) *fake.Clientset {
		return fake.NewSimpleClientset(&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Spec: configv1.ClusterVersionSpec{
				Channel: channel,
			},
			Status: configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: version,
				},
			},
		})
	}

	for _, tt := range []struct {
		name           string
		fakecli        *fake.Clientset
		desiredVersion string
		wantUpdated    bool
		wantErr        string
	}{
		{
			name:        "non-existing version - no update",
			fakecli:     newFakecli("", "0.0.0"),
			wantUpdated: false,
		},
		{
			name:    "right version, no update needed",
			fakecli: newFakecli("", Stream44.Version.String()),
		},
		{
			name:    "higher version, no update needed",
			fakecli: newFakecli("", "4.4.5"),
		},
		{
			name:           "lower version, update needed (4.4)",
			fakecli:        newFakecli("", "4.4.1"),
			wantUpdated:    true,
			desiredVersion: Stream44.Version.String(),
		},
		{
			name:    "higher version, no update needed",
			fakecli: newFakecli("", "4.3.19"),
		},
		{
			name:           "lower version, update needed (3.3)",
			fakecli:        newFakecli("", "4.3.14"),
			wantUpdated:    true,
			desiredVersion: Stream43.Version.String(),
		},
		{
			name:           "on a channel, update needed",
			fakecli:        newFakecli("my-channel", "4.3.14"),
			desiredVersion: Stream43.Version.String(),
			wantUpdated:    true,
		},
		{
			name:           "on a channel, no update needed",
			fakecli:        newFakecli("my-channel", "4.4.4"),
			desiredVersion: Stream44.Version.String(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var updated bool

			tt.fakecli.PrependReactor("update", "clusterversions", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			i := &Installer{
				log:       logrus.NewEntry(logrus.StandardLogger()),
				configcli: tt.fakecli,
			}

			err := i.upgradeCluster(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if updated != tt.wantUpdated {
				t.Fatal(updated)
			}

			cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if tt.wantUpdated {
				if cv.Spec.DesiredUpdate == nil {
					t.Fatal(cv.Spec.DesiredUpdate)
				}
				if cv.Spec.DesiredUpdate.Version != tt.desiredVersion {
					t.Error(cv.Spec.DesiredUpdate.Version)
				}
			}
		})
	}
}
