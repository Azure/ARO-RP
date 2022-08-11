package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	HIVEENVVARIABLE = "HIVEKUBECONFIGPATH"
)

func hiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	// Indexes above 1 have _index appended to them
	envVar := HIVEENVVARIABLE
	if index > 1 {
		envVar = fmt.Sprintf("%s_%d", HIVEENVVARIABLE, index)
	}

	kubeConfigPath := os.Getenv(envVar)
	if kubeConfigPath == "" {
		return nil, fmt.Errorf("missing %s env variable", HIVEENVVARIABLE)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func (d *dev) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	return hiveRestConfig(ctx, index)
}

// pass through env var for now
func (p *prod) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	return hiveRestConfig(ctx, index)
}
