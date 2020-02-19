package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Azure/ARO-RP/pkg/entrypoint/mirror"
	"github.com/Azure/ARO-RP/pkg/entrypoint/monitor"
	"github.com/Azure/ARO-RP/pkg/entrypoint/rp"
)

var gitCommit = "unknown"

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	rootCmd := &cobra.Command{
		Use:  "./aro [component]",
		Long: "Azure Red Hat OpenShift V4 dispatcher",
	}
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringP("loglevel", "l", "Debug", "Valid values are [panic,fatal,error,warning,info,debug,trace]")
	rootCmd.Printf("gitCommit %s\n", gitCommit)

	rootCmd.AddCommand(mirror.NewCommand())
	rootCmd.AddCommand(monitor.NewCommand())
	rootCmd.AddCommand(rp.NewCommand())

	return rootCmd.Execute()
}
