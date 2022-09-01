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
	}
	return nil, errors.New("testLiveConfig does not have a Hive")
}

func (t *testLiveConfig) InstallViaHive(ctx context.Context) (bool, error) {
	return t.hasHive, nil
}

func (t *testLiveConfig) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	if t.hasHive {
		return "example/pull:spec"
	}
	return ""
}

func NewTestLiveConfig(hasHive bool) liveconfig.Manager {
	return &testLiveConfig{hasHive: hasHive}
}
