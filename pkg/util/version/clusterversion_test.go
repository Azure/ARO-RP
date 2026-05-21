package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestGetClusterVersion(t *testing.T) {
	for _, tt := range []struct {
		name     string
		wantVers Version
		mocks    func(*configv1.ClusterVersion)
		wantErr  string
	}{
		{
			name: "cluster version nil returns error",
			//nolint:staticcheck // Ignore: SA4009 argument cv is overwritten before first use (staticcheck)
			mocks: func(cv *configv1.ClusterVersion) {
				cv = nil
			},
			wantErr: "unknown cluster version",
		},
		{
			name:    "no update history returns error",
			wantErr: "unknown cluster version",
		},
		{
			name:     "multiple completed updates returns top most",
			wantVers: NewVersion(4, 10, 0),
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = append(cv.Status.History,
					configv1.UpdateHistory{
						State:   configv1.CompletedUpdate,
						Version: "4.10.0",
					},
					configv1.UpdateHistory{
						State:   configv1.CompletedUpdate,
						Version: "4.9.0",
					},
				)
			},
		},
		{
			name:     "pending update topmost, returns most recent completed update",
			wantVers: NewVersion(4, 9, 0),
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = append(cv.Status.History,
					configv1.UpdateHistory{
						State:   configv1.PartialUpdate,
						Version: "4.10.0",
					},
					configv1.UpdateHistory{
						State:   configv1.CompletedUpdate,
						Version: "4.9.0",
					},
				)
			},
		},
		{
			name:     "only partial update in history returns partial update",
			wantVers: NewVersion(4, 10, 0),
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = append(cv.Status.History,
					configv1.UpdateHistory{
						State:   configv1.PartialUpdate,
						Version: "4.10.0",
					},
				)
			},
		},
		{
			name:     "missing update history state and no completed returns version",
			wantVers: NewVersion(4, 10, 0),
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = append(cv.Status.History,
					configv1.UpdateHistory{
						Version: "4.10.0",
					},
				)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cv := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{},
				},
			}

			if tt.mocks != nil {
				tt.mocks(cv)
			}
			version, err := GetClusterVersion(cv)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !reflect.DeepEqual(version, tt.wantVers) {
				t.Error(version)
			}
		})
	}
}

func TestIsClusterUpgrading(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mocks   func(*configv1.ClusterVersion)
		nilCV   bool
		want    bool
		comment string
	}{
		{
			name:    "nil ClusterVersion returns false",
			nilCV:   true,
			want:    false,
			comment: "Safety check for nil input",
		},
		{
			name:    "empty history returns false",
			mocks:   nil,
			want:    false,
			comment: "Fresh cluster or no upgrade history",
		},
		{
			name: "steady state - Completed update, Progressing=False returns false",
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.30",
					},
				}
				cv.Status.Conditions = []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorProgressing,
						Status: configv1.ConditionFalse,
					},
				}
			},
			want:    false,
			comment: "Normal steady state cluster",
		},
		{
			name: "ARO-26990 bug case - Completed update but Progressing=True returns false",
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.30",
					},
				}
				cv.Status.Conditions = []configv1.ClusterOperatorStatusCondition{
					{
						Type:    configv1.OperatorProgressing,
						Status:  configv1.ConditionTrue,
						Reason:  "ClusterOperatorProgressing",
						Message: "Working towards 4.18.30: some cluster operators are still updating",
					},
				}
			},
			want:    false,
			comment: "MCO rollout from ARO MC deploy - should NOT trigger dnsmasq updates",
		},
		{
			name: "upgrade intent not initiated - desiredUpdate set but history[0] still Completed returns false",
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Spec.DesiredUpdate = &configv1.Update{
					Version: "4.19.0",
				}
				cv.Status.History = []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.30",
					},
				}
				cv.Status.Conditions = []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorProgressing,
						Status: configv1.ConditionFalse,
					},
				}
			},
			want:    false,
			comment: "Customer expressed upgrade intent but CVO hasn't started (e.g., pending adminack)",
		},
		{
			name: "upgrade initiated - Partial update at history[0] returns true",
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Spec.DesiredUpdate = &configv1.Update{
					Version: "4.19.0",
				}
				cv.Status.History = []configv1.UpdateHistory{
					{
						State:   configv1.PartialUpdate,
						Version: "4.19.0",
					},
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.30",
					},
				}
				cv.Status.Conditions = []configv1.ClusterOperatorStatusCondition{
					{
						Type:    configv1.OperatorProgressing,
						Status:  configv1.ConditionTrue,
						Reason:  "ClusterOperatorProgressing",
						Message: "Working towards 4.19.0",
					},
				}
			},
			want:    true,
			comment: "OCP upgrade in progress - dnsmasq updates allowed",
		},
		{
			name: "upgrade initiated - no State field (treated as non-Completed) returns true",
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = []configv1.UpdateHistory{
					{
						Version: "4.19.0",
						// State field missing - older clusters or edge cases
					},
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.30",
					},
				}
			},
			want:    true,
			comment: "Missing State field treated as upgrade in progress",
		},
		{
			name: "multiple completed updates returns false",
			mocks: func(cv *configv1.ClusterVersion) {
				cv.Status.History = []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.30",
					},
					{
						State:   configv1.CompletedUpdate,
						Version: "4.18.0",
					},
					{
						State:   configv1.CompletedUpdate,
						Version: "4.17.38",
					},
				}
			},
			want:    false,
			comment: "Cluster with upgrade history, currently at steady state",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var cv *configv1.ClusterVersion

			if !tt.nilCV {
				cv = &configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Status: configv1.ClusterVersionStatus{
						History:    []configv1.UpdateHistory{},
						Conditions: []configv1.ClusterOperatorStatusCondition{},
					},
				}

				if tt.mocks != nil {
					tt.mocks(cv)
				}
			}

			got := IsClusterUpgrading(cv)
			if got != tt.want {
				t.Errorf("IsClusterUpgrading() = %v, want %v (%s)", got, tt.want, tt.comment)
			}
		})
	}
}
