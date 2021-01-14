package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
)

func TestRHCOSImage(t *testing.T) {
	ctx := context.Background()

	i, err := getRHCOSImage(ctx)
	if err != nil {
		t.Error(err)
	}

	if i == nil {
		t.Error(i)
	}
}
