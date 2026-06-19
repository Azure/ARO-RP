package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/mimo"
)

func TestDefaultMaintenanceTasksIncludesOperatorImageAutoUpdate(t *testing.T) {
	if _, ok := DEFAULT_MAINTENANCE_TASKS[mimo.OPERATOR_IMAGE_AUTO_UPDATE_ID]; !ok {
		t.Fatalf("task map missing %q", mimo.OPERATOR_IMAGE_AUTO_UPDATE_ID)
	}
}

