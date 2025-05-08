package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "embed"
)

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
