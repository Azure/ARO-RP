package testliveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

type testLiveConfig struct {
	adoptByHive      bool
	installViaHive   bool
	installerBackend api.InstallerBackend
}

func (t *testLiveConfig) HiveRestConfig(ctx context.Context, shard int) (*rest.Config, error) {
	if t.adoptByHive || t.installViaHive {
		return &rest.Config{}, nil
	}
	return nil, errors.New("testLiveConfig does not have a Hive")
}

func (t *testLiveConfig) InstallerBackend(ctx context.Context) (api.InstallerBackend, error) {
	// If explicitly set, use that
	if t.installerBackend != "" {
		return t.installerBackend, nil
	}
	// Otherwise use legacy installViaHive flag
	if t.installViaHive {
		return api.InstallerBackendHive, nil
	}
	return api.InstallerBackendPodman, nil
}

func (t *testLiveConfig) InstallViaHive(ctx context.Context) (bool, error) {
	return t.installViaHive, nil
}

func (t *testLiveConfig) AdoptByHive(ctx context.Context) (bool, error) {
	return t.adoptByHive, nil
}

func (t *testLiveConfig) InstallerRestConfig(ctx context.Context) (*rest.Config, error) {
	backend, err := t.InstallerBackend(ctx)
	if err != nil {
		return nil, err
	}

	if backend == api.InstallerBackendPodman {
		return nil, errors.New("podman backend does not use Kubernetes")
	}

	// For test purposes, return the same config as Hive
	return t.HiveRestConfig(ctx, 0)
}

func (t *testLiveConfig) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	if t.installViaHive {
		return "example/pull:spec"
	}
	return ""
}

func NewTestLiveConfig(adoptByHive, installViaHive bool) liveconfig.Manager {
	backend := api.InstallerBackendPodman
	if installViaHive {
		backend = api.InstallerBackendHive
	}
	return &testLiveConfig{
		adoptByHive:      adoptByHive,
		installViaHive:   installViaHive,
		installerBackend: backend,
	}
}

func NewTestLiveConfigWithBackend(adoptByHive bool, backend api.InstallerBackend) liveconfig.Manager {
	return &testLiveConfig{
		adoptByHive:      adoptByHive,
		installViaHive:   backend == api.InstallerBackendHive,
		installerBackend: backend,
	}
}
