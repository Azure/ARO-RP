package rp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/Azure/ARO-RP/pkg/entrypoint/config"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

type Config struct {
	config.Common
}

// NewCommand returns the cobra command for "rp".
func NewCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:  "rp",
		Long: "Start ARO RP",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig(cmd)
			if err != nil {
				return err
			}

			ctx := context.Background()
			log := utillog.GetLogger(cfg.LogLevel)

			return start(ctx, log, cfg)
		},
	}

	return cc
}

func getConfig(cmd *cobra.Command) (*Config, error) {
	var c Config
	var err error
	c.Common, err = config.CommonConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
