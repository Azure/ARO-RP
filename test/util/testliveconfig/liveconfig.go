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
	adoptByHive    bool
	installViaHive bool
	useCheckAccess bool
}

func (t *testLiveConfig) HiveRestConfig(ctx context.Context, shard int) (*rest.Config, error) {
	if t.adoptByHive || t.installViaHive {
		return &rest.Config{}, nil
	}
	return nil, errors.New("testLiveConfig does not have a Hive")
}

func (t *testLiveConfig) InstallViaHive(ctx context.Context) (bool, error) {
	return t.installViaHive, nil
}

func (t *testLiveConfig) AdoptByHive(ctx context.Context) (bool, error) {
	return t.adoptByHive, nil
}

func (t *testLiveConfig) UseCheckAccess(ctx context.Context) (bool, error) {
	return t.useCheckAccess, nil
}

func (t *testLiveConfig) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	if t.installViaHive {
		return "example/pull:spec"
	}
	return ""
}

func NewTestLiveConfig(adoptByHive, installViaHive, useCheckAccess bool) liveconfig.Manager {
	return &testLiveConfig{
		adoptByHive:    adoptByHive,
		installViaHive: installViaHive,
		useCheckAccess: useCheckAccess,
	}
}
