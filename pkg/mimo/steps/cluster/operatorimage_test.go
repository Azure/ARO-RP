package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_deploy "github.com/Azure/ARO-RP/pkg/util/mocks/operator/deploy"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestUpdateAROOperatorImage(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		err     error
		wantErr string
	}{
		{
			name: "updates operator image",
		},
		{
			name:    "returns transient update error",
			err:     errors.New("update failed"),
			wantErr: "TransientError: failed to update ARO operator image: update failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockEnv := mock_env.NewMockInterface(controller)
			deployer := mock_deploy.NewMockOperator(controller)
			deployer.EXPECT().Update(gomock.Any()).Return(tt.err)

			_, log := testlog.New()
			tc := testtasks.NewFakeTestContext(
				ctx,
				mockEnv,
				log,
				func() time.Time { return time.Unix(100, 0) },
				testtasks.WithAROOperatorDeployer(deployer),
			)

			err := UpdateAROOperatorImage(tc)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("unexpected error: got %q want %q", err.Error(), tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAROOperatorDeploymentReady(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		ready   bool
		err     error
		want    bool
		wantErr string
	}{
		{
			name:  "ready",
			ready: true,
			want:  true,
		},
		{
			name: "not ready",
		},
		{
			name:    "returns transient readiness error",
			err:     errors.New("readiness failed"),
			wantErr: "TransientError: failed to check ARO operator deployment readiness: readiness failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockEnv := mock_env.NewMockInterface(controller)
			deployer := mock_deploy.NewMockOperator(controller)
			deployer.EXPECT().IsReady(gomock.Any()).Return(tt.ready, tt.err)

			_, log := testlog.New()
			tc := testtasks.NewFakeTestContext(
				ctx,
				mockEnv,
				log,
				func() time.Time { return time.Unix(100, 0) },
				testtasks.WithAROOperatorDeployer(deployer),
			)

			got, err := AROOperatorDeploymentReady(tc)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("unexpected error: got %q want %q", err.Error(), tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected readiness: got %t want %t", got, tt.want)
			}
		})
	}
}

func TestEnsureAROOperatorRunningDesiredVersion(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	mockEnv := mock_env.NewMockInterface(controller)
	deployer := mock_deploy.NewMockOperator(controller)
	deployer.EXPECT().IsRunningDesiredVersion(gomock.Any()).Return(true, nil)

	_, log := testlog.New()
	tc := testtasks.NewFakeTestContext(
		ctx,
		mockEnv,
		log,
		func() time.Time { return time.Unix(100, 0) },
		testtasks.WithAROOperatorDeployer(deployer),
	)

	got, err := EnsureAROOperatorRunningDesiredVersion(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected desired version")
	}
}

func TestSyncClusterObject(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	mockEnv := mock_env.NewMockInterface(controller)
	mockEnv.EXPECT().AROOperatorImage().Return("example.azurecr.io/aro:v20250102.1")

	deployer := mock_deploy.NewMockOperator(controller)
	deployer.EXPECT().SyncClusterObject(gomock.Any()).Return(nil)

	_, log := testlog.New()
	tc := testtasks.NewFakeTestContext(
		ctx,
		mockEnv,
		log,
		func() time.Time { return time.Unix(100, 0) },
		testtasks.WithAROOperatorDeployer(deployer),
	)

	err := SyncClusterObject(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(tc.GetResultMessage(), "example.azurecr.io/aro:v20250102.1") {
		t.Fatalf("unexpected result message: %q", tc.GetResultMessage())
	}
}
