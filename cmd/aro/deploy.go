package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	deployer "github.com/Azure/ARO-RP/pkg/deploy"
)

type mode int

const (
	modePreDeploy mode = iota
	modeDeploy
	modeUpgrade
	modeFull
)

type flags struct {
	configFile    string
	deployVersion string
	location      string
	mode          string
}

func deploy(ctx context.Context, log *logrus.Entry) error {
	f, err := parseFlags()
	if err != nil {
		return err
	}

	config, err := deployer.GetConfig(f.configFile, f.location)
	if err != nil {
		return err
	}

	log.Infof("FullDeploy mode status: %t", strings.Contains(f.mode, "f"))
	d, err := deployer.New(ctx, log, config, f.deployVersion, strings.Contains(f.mode, "f"))
	if err != nil {
		return err
	}

	if strings.Contains(f.mode, "p") {
		err := d.PreDeploy(ctx)
		if err != nil {
			return err
		}
	}
	if strings.Contains(f.mode, "d") {
		err := d.Deploy(ctx)
		if err != nil {
			return err
		}
	}
	if strings.Contains(f.mode, "u") {
		err := d.Upgrade(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseFlags() (*flags, error) {
	deployVersion, location := gitCommit, flag.Arg(3)

	if os.Getenv("RP_VERSION") != "" {
		deployVersion = os.Getenv("RP_VERSION")
	}

	if deployVersion == "unknown" || strings.Contains(deployVersion, "dirty") {
		return nil, fmt.Errorf("invalid deploy version %q", deployVersion)
	}

	if strings.ToLower(location) != location {
		return nil, fmt.Errorf("location %s must be lower case", location)
	}

	return &flags{
		location:      location,
		deployVersion: deployVersion,
		configFile:    flag.Arg(1),
		mode:          flag.Arg(2),
	}, nil
}
