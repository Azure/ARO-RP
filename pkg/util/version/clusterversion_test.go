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
		wantVers *Version
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
