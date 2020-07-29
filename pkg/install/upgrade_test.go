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

	stream43 := version.Stream{
		Version: version.NewVersion(4, 3, 27),
	}
	stream44 := version.Stream{
		Version: version.NewVersion(4, 4, 10),
	}
	stream45 := version.Stream{
		Version: version.NewVersion(4, 5, 3),
	}

	newFakecli := func(status configv1.ClusterVersionStatus) *fake.Clientset {
		return fake.NewSimpleClientset(&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Spec: configv1.ClusterVersionSpec{
				Channel: "",
			},
			Status: status,
		})
	}

	for _, tt := range []struct {
		name    string
		fakecli *fake.Clientset

		desiredVersion string
		wantUpdated    bool
		wantErr        string
	}{
		{
			name: "unhealthy cluster",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
					},
				},
			}),
			wantUpdated: false,
			wantErr:     "not upgrading: previous upgrade in-progress",
		},
		{
			name: "upgrade to Y latest",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: "4.3.1",
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			wantUpdated:    true,
			desiredVersion: stream43.Version.String(),
		},
		{
			name: "no upgrade, Y higher than expected",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: "4.3.99",
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			wantUpdated: false,
			wantErr:     "not upgrading: cvo desired version is 4.3.99",
		},
		{
			name: "no upgrade, Y match but unhealthy cluster",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
					},
				},
			}),
			wantUpdated: false,
			wantErr:     "not upgrading: previous upgrade in-progress",
		},
		{
			name: "upgrade, Y match",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			wantUpdated:    true,
			desiredVersion: stream44.Version.String(),
		},
		{
			name: "upgrade, Y match 2",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Update{
					Version: stream44.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			wantUpdated:    true,
			desiredVersion: stream45.Version.String(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			version.Streams = append([]version.Stream{}, stream43, stream44, stream45)
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
