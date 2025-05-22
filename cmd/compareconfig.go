package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/yaml"

	"github.com/Azure/ARO-RP/pkg/deploy"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func main() {
	opts := DefaultOptions()
	cmd := &cobra.Command{
		Use:          "overview",
		Short:        "overview",
		Long:         "overview",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return compare(cmd.Context(), opts)
		},
	}
	if err := BindOptions(opts, cmd); err != nil {
		slog.Error("failed to bind options", "error", err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		slog.Error("failed to compare configs", "error", err)
		os.Exit(1)
	}
}

func compare(ctx context.Context, opts *RawOptions) error {
	validated, err := opts.Validate()
	if err != nil {
		return err
	}
	completed, err := validated.Complete()
	if err != nil {
		return err
	}
	if diff := cmp.Diff(completed.ClassicConfig, completed.RAConfig); diff != "" {
		fmt.Println(diff)
		return fmt.Errorf("configs are different")
	}
	return nil
}

func DefaultOptions() *RawOptions {
	return &RawOptions{}
}

func BindOptions(opts *RawOptions, cmd *cobra.Command) error {
	cmd.Flags().StringVar(&opts.ClassicConfigDir, "classic-config-dir", opts.ClassicConfigDir, "path to directory holding classic configurations")
	cmd.Flags().StringVar(&opts.RAConfigDir, "ra-config-dir", opts.RAConfigDir, "path to directory holding RA configurations")

	for _, flag := range []string{"classic-config-dir", "ra-config-dir"} {
		if err := cmd.MarkFlagDirname(flag); err != nil {
			return fmt.Errorf("failed to mark flag %q as a directory: %w", flag, err)
		}
	}
	return nil
}

// RawOptions holds input values.
type RawOptions struct {
	ClassicConfigDir string
	RAConfigDir      string
}

// validatedOptions is a private wrapper that enforces a call of Validate() before Complete() can be invoked.
type validatedOptions struct {
	*RawOptions
}

type ValidatedOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*validatedOptions
}

// completedOptions is a private wrapper that enforces a call of Complete() before config generation can be invoked.
type completedOptions struct {
	// ClassicConfig stores cloud->environment->region->config
	ClassicConfig map[string]map[string]map[string]*deploy.RPConfig
	// RAConfig stores cloud->environment->region->config
	RAConfig map[string]map[string]map[string]*deploy.RPConfig
}

type Options struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedOptions
}

func (o *RawOptions) Validate() (*ValidatedOptions, error) {
	if o.ClassicConfigDir == "" {
		return nil, fmt.Errorf("directory holding classic config is required")
	}

	if o.RAConfigDir == "" {
		return nil, fmt.Errorf("directory holding classic config is required")
	}

	return &ValidatedOptions{
		validatedOptions: &validatedOptions{
			RawOptions: o,
		},
	}, nil
}

func (o *ValidatedOptions) Complete() (*Options, error) {
	classicConfig := make(map[string]map[string]map[string]*deploy.RPConfig)
	if err := filepath.WalkDir(o.ClassicConfigDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), "ra") {
			return nil
		}
		cloud := "public"
		if strings.HasPrefix(d.Name(), "ff") {
			cloud = "ff"
		}
		if _, exists := classicConfig[cloud]; !exists {
			classicConfig[cloud] = make(map[string]map[string]*deploy.RPConfig)
		}
		env := strings.TrimSuffix(strings.TrimPrefix(d.Name(), "ff"), "-config.yaml")
		if _, exists := classicConfig[cloud][env]; !exists {
			classicConfig[cloud][env] = make(map[string]*deploy.RPConfig)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read classic config[%s][%s] file %s: %w", cloud, env, path, err)
		}
		var config deploy.Config
		if err := yaml.Unmarshal(raw, &config); err != nil {
			return fmt.Errorf("failed to unmarshal classic config[%s][%s] file %s: %w", cloud, env, path, err)
		}
		for _, region := range config.RPs {
			slog.Info("resolving classic config", "cloud", cloud, "env", env, "region", region.Location)
			rpConfig, err := deploy.ResolveConfig(&config, region.Location)
			if err != nil {
				return fmt.Errorf("failed to resolve classic config[%s][%s][%s]: %w", cloud, env, region.Location, err)
			}
			if _, exists := classicConfig[cloud][env][region.Location]; exists {
				return fmt.Errorf("found duplicate classic region config[%s][%s][%s]", cloud, env, region.Location)
			}
			classicConfig[cloud][env][region.Location] = rpConfig
		}
		return nil
	}); err != nil {
		return nil, err
	}

	raConfig := make(map[string]map[string]map[string]*deploy.RPConfig)
	if err := filepath.WalkDir(o.RAConfigDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		parts := strings.Split(strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())), ".")
		if len(parts) != 3 {
			return nil
		}
		cloud, env, region := parts[0], parts[1], parts[2]
		if _, exists := raConfig[cloud]; !exists {
			raConfig[cloud] = make(map[string]map[string]*deploy.RPConfig)
		}
		if _, exists := raConfig[cloud][env]; !exists {
			raConfig[cloud][env] = make(map[string]*deploy.RPConfig)
		}

		slog.Info("resolving ra config", "cloud", cloud, "env", env, "region", region)
		rpConfig, err := deploy.GetConfig(path, region)
		if err != nil {
			return fmt.Errorf("failed to resolve classic config[%s][%s][%s]: %w", cloud, env, region, err)
		}
		if _, exists := raConfig[cloud][env][region]; exists {
			return fmt.Errorf("found duplicate classic region config[%s][%s][%s]", cloud, env, region)
		}
		raConfig[cloud][env][region] = rpConfig
		return nil
	}); err != nil {
		return nil, err
	}

	return &Options{
		completedOptions: &completedOptions{
			ClassicConfig: classicConfig,
			RAConfig:      raConfig,
		},
	}, nil
}
