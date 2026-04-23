package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestDeleteRoleAssignmentsSkipsServicePrincipalCleanupForWorkloadIdentity(t *testing.T) {
	c := &Cluster{
		log:    logrus.NewEntry(logrus.New()),
		Config: &ClusterConfig{UseWorkloadIdentity: true},
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("deleteRoleAssignments panicked instead of short-circuiting: %v", recovered)
		}
	}()

	if err := c.deleteRoleAssignments(context.Background(), "test-rg", "test-cluster"); err != nil {
		t.Fatalf("deleteRoleAssignments() error = %v, want nil", err)
	}
}
