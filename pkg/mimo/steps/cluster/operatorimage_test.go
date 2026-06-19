package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestLatestOperatorImageVersionFromTags(t *testing.T) {
	got, ok := latestOperatorImageVersionFromTags([]string{
		"latest",
		"v20250101.2",
		"v20250101.10",
		"bad",
		"v20241231.99",
	})
	if !ok {
		t.Fatal("expected to find a valid operator image version")
	}

	if got.raw != "v20250101.10" {
		t.Fatalf("unexpected latest version: %q", got.raw)
	}
}

func TestLatestOperatorImageVersionFromTagsNoValidTags(t *testing.T) {
	_, ok := latestOperatorImageVersionFromTags([]string{"latest", "dev", "abc"})
	if ok {
		t.Fatal("expected no valid operator image versions")
	}
}

func TestListACRRepositoryTagsPagination(t *testing.T) {
	const username = "testuser"
	const password = "testpass"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != username || p != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		last := r.URL.Query().Get("last")
		if last == "" {
			w.Header().Set("Link", `</v2/aro/tags/list?n=1000&last=v20250101.0>; rel="next"`)
			_, _ = w.Write([]byte(`{"name":"aro","tags":["v20250101.0","dev"]}`))
			return
		}

		_, _ = w.Write([]byte(`{"name":"aro","tags":["v20250102.3"]}`))
	}))
	defer server.Close()

	tags, err := listACRRepositoryTags(
		context.Background(),
		server.Client(),
		server.Listener.Addr().String(),
		"aro",
		username,
		password,
	)
	if err != nil {
		t.Fatalf("unexpected error listing tags: %v", err)
	}

	want := []string{"v20250101.0", "dev", "v20250102.3"}
	if strings.Join(tags, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected tags: got %v want %v", tags, want)
	}
}

func TestEnsureOperatorImageUpdateScheduled(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                    string
		operatorVersion         string
		maintenanceTask         api.MaintenanceTask
		registryProfiles        []*api.RegistryProfile
		tags                    []string
		listErr                 error
		wantErr                 string
		wantOperatorVersion     string
		wantMaintenanceTask     api.MaintenanceTask
		wantResultMessageSubstr string
	}{
		{
			name:                    "schedules operator update when newer version exists",
			operatorVersion:         "v20250101.0",
			maintenanceTask:         api.MaintenanceTaskNone,
			registryProfiles:        []*api.RegistryProfile{{Name: "example.azurecr.io", Username: "u", Password: api.SecureString("p")}},
			tags:                    []string{"v20250101.0", "v20250102.1"},
			wantOperatorVersion:     "v20250102.1",
			wantMaintenanceTask:     api.MaintenanceTaskOperator,
			wantResultMessageSubstr: "scheduled operator image update",
		},
		{
			name:                    "no-op when already up to date",
			operatorVersion:         "v20250102.1",
			maintenanceTask:         api.MaintenanceTaskNone,
			registryProfiles:        []*api.RegistryProfile{{Name: "example.azurecr.io", Username: "u", Password: api.SecureString("p")}},
			tags:                    []string{"v20250101.0", "v20250102.1"},
			wantOperatorVersion:     "v20250102.1",
			wantMaintenanceTask:     api.MaintenanceTaskNone,
			wantResultMessageSubstr: "already up to date",
		},
		{
			name:                    "skips when maintenance task already active",
			operatorVersion:         "v20250101.0",
			maintenanceTask:         api.MaintenanceTaskEverything,
			registryProfiles:        []*api.RegistryProfile{{Name: "example.azurecr.io", Username: "u", Password: api.SecureString("p")}},
			tags:                    []string{"v20250102.1"},
			wantOperatorVersion:     "v20250101.0",
			wantMaintenanceTask:     api.MaintenanceTaskEverything,
			wantResultMessageSubstr: "skipping operator image auto-update",
		},
		{
			name:                    "skips unmanaged current operator version value",
			operatorVersion:         "override",
			maintenanceTask:         api.MaintenanceTaskNone,
			registryProfiles:        []*api.RegistryProfile{{Name: "example.azurecr.io", Username: "u", Password: api.SecureString("p")}},
			tags:                    []string{"v20250102.1"},
			wantOperatorVersion:     "override",
			wantMaintenanceTask:     api.MaintenanceTaskNone,
			wantResultMessageSubstr: "not managed by this task",
		},
		{
			name:                "fails without registry profile credentials",
			operatorVersion:     "v20250101.0",
			maintenanceTask:     api.MaintenanceTaskNone,
			registryProfiles:    nil,
			tags:                []string{"v20250102.1"},
			wantErr:             "TerminalError: missing registry profile credentials for operator image updates",
			wantOperatorVersion: "v20250101.0",
			wantMaintenanceTask: api.MaintenanceTaskNone,
		},
		{
			name:                "propagates transient tag listing error",
			operatorVersion:     "v20250101.0",
			maintenanceTask:     api.MaintenanceTaskNone,
			registryProfiles:    []*api.RegistryProfile{{Name: "example.azurecr.io", Username: "u", Password: api.SecureString("p")}},
			listErr:             mimo.TransientError(errors.New("temporary registry outage")),
			wantErr:             "TransientError: temporary registry outage",
			wantOperatorVersion: "v20250101.0",
			wantMaintenanceTask: api.MaintenanceTaskNone,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			originalListFn := listACRRepositoryTagsFn
			listACRRepositoryTagsFn = func(ctx context.Context, client *http.Client, registryHost, repository, username, password string) ([]string, error) {
				if tt.listErr != nil {
					return nil, tt.listErr
				}
				return tt.tags, nil
			}
			defer func() {
				listACRRepositoryTagsFn = originalListFn
			}()

			controller := gomock.NewController(t)
			defer controller.Finish()

			mockEnv := mock_env.NewMockInterface(controller)
			mockEnv.EXPECT().ACRDomain().AnyTimes().Return("example.azurecr.io")

			_, log := testlog.New()
			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()

			resourceID := strings.ToLower(testdatabase.GetResourcePath("00000000-0000-0000-0000-000000000000", "testcluster"))
			doc := &api.OpenShiftClusterDocument{
				ID:  "test-doc",
				Key: resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: resourceID,
					Properties: api.OpenShiftClusterProperties{
						OperatorVersion:  tt.operatorVersion,
						MaintenanceTask:  tt.maintenanceTask,
						RegistryProfiles: tt.registryProfiles,
					},
				},
			}

			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(doc)
			if err := fixture.Create(); err != nil {
				t.Fatal(err)
			}

			tc := testtasks.NewFakeTestContext(
				ctx,
				mockEnv,
				log,
				func() time.Time { return time.Unix(100, 0) },
				testtasks.WithOpenShiftClusterDocument(doc),
				testtasks.WithOpenShiftDatabase(openShiftClustersDatabase),
			)

			err := EnsureOperatorImageUpdateScheduled(tc)
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

			saved, err := openShiftClustersDatabase.Get(ctx, resourceID)
			if err != nil {
				t.Fatal(err)
			}
			if saved.OpenShiftCluster.Properties.OperatorVersion != tt.wantOperatorVersion {
				t.Fatalf("unexpected operatorVersion: got %q want %q", saved.OpenShiftCluster.Properties.OperatorVersion, tt.wantOperatorVersion)
			}
			if saved.OpenShiftCluster.Properties.MaintenanceTask != tt.wantMaintenanceTask {
				t.Fatalf("unexpected maintenanceTask: got %q want %q", saved.OpenShiftCluster.Properties.MaintenanceTask, tt.wantMaintenanceTask)
			}
			if tt.wantResultMessageSubstr != "" && !strings.Contains(tc.GetResultMessage(), tt.wantResultMessageSubstr) {
				t.Fatalf("unexpected result message: got %q expected substring %q", tc.GetResultMessage(), tt.wantResultMessageSubstr)
			}
		})
	}
}

