package testliveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

type testLiveConfig struct {
	hasHive bool
}

func (t *testLiveConfig) HiveRestConfig(ctx context.Context, shard int) (*rest.Config, error) {
	if t.hasHive {
		return &rest.Config{}, nil
	} else {
		return nil, errors.New("testLiveConfig does not have a Hive")
	}
}

func (t *testLiveConfig) InstallViaHive(ctx context.Context) (bool, error) {
	return t.hasHive, nil
}

func (t *testLiveConfig) DefaultInstallerPullSpec(ctx context.Context) (string, error) {
	if t.hasHive {
		return "example/pull:spec", nil
	} else {
		return "", nil
	}
}

func NewTestLiveConfig(hasHive bool) liveconfig.Manager {
	return &testLiveConfig{hasHive: hasHive}
}
