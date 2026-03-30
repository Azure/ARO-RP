package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	_ "embed"
)

// stripShellComments removes comment-only lines and blank lines from shell
// scripts to reduce the size of the base64-encoded ARM template expression.
// Shebangs (#!/...) are preserved. This is necessary because the Azure ARM
// template language expression limit is 81,920 characters, and our concatenated
// bootstrap scripts exceed that when left unstripped.
func stripShellComments(script string) string {
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "#!") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

//go:embed scripts/devProxyVMSS.sh
var scriptDevProxyVMSS string

//go:embed scripts/gatewayVMSS.sh
var scriptGatewayVMSS string

//go:embed scripts/rpVMSS.sh
var scriptRpVMSS string

//go:embed scripts/util-system.sh
var scriptUtilSystem string

//go:embed scripts/util-services.sh
var scriptUtilServices string

//go:embed scripts/util-packages.sh
var scriptUtilPackages string

//go:embed scripts/util-common.sh
var scriptUtilCommon string
