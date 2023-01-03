package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

func TestUpgradeCluster(t *testing.T) {
	ctx := context.Background()

	stream43 := &version.Stream{
		Version: version.NewVersion(4, 3, 27),
	}
	stream44 := &version.Stream{
		Version: version.NewVersion(4, 4, 10),
	}
	stream45 := &version.Stream{
		Version: version.NewVersion(4, 5, 3),
	}

	newFakecli := func(status configv1.ClusterVersionStatus) *configfake.Clientset {
		return configfake.NewSimpleClientset(&configv1.ClusterVersion{
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
		fakecli *configfake.Clientset

		desiredVersion string
		upgradeY       bool
		wantUpdated    bool
		wantErr        string
	}{
		{
			name: "unhealthy cluster",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
					},
				},
			}),
			wantErr: "500: InternalServerError: : Not upgrading: cvo is unhealthy.",
		},
		{
			name: "upgrade to Y latest",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: "4.3.1",
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			desiredVersion: stream43.Version.String(),
			wantUpdated:    true,
		},
		{
			name: "no upgrade, Y higher than expected",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: "4.3.99",
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
		},
		{
			name: "no upgrade, Y match but unhealthy cluster",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
					},
				},
			}),
			wantErr: "500: InternalServerError: : Not upgrading: cvo is unhealthy.",
		},
		{
			name: "upgrade, Y match, Y upgrades NOT allowed",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
		},
		{
			name: "upgrade, Y match, Y upgrades allowed (4.3 to 4.4)",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: stream43.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			desiredVersion: stream44.Version.String(),
			upgradeY:       true,
			wantUpdated:    true,
		},
		{
			name: "upgrade, Y match, Y upgrades allowed (4.4 to 4.5)",
			fakecli: newFakecli(configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: stream44.Version.String(),
				},
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionTrue,
					},
				},
			}),
			desiredVersion: stream45.Version.String(),
			upgradeY:       true,
			wantUpdated:    true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var updated bool

			tt.fakecli.PrependReactor("update", "clusterversions", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			k := &kubeActions{
				log:       logrus.NewEntry(logrus.StandardLogger()),
				configcli: tt.fakecli,
			}

			err := upgrade(ctx, k.log, k.configcli, []*version.Stream{stream43, stream44, stream45}, tt.upgradeY)
			matcher.AssertErrHasWantMsg(t, err, tt.wantErr)

			if updated != tt.wantUpdated {
				t.Fatal(updated)
			}

			cv, err := k.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
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
