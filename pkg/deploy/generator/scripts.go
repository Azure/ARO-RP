package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "embed"
)

//go:embed scripts/devProxyVMSS.sh
var scriptDevProxyVMSS []byte

//go:embed scripts/gatewayVMSS.sh
var scriptGatewayVMSS []byte

//go:embed scripts/rpVMSS.sh
var scriptRpVMSS []byte

//go:embed scripts/util-system.sh
var scriptUtilSystem []byte

//go:embed scripts/util-services.sh
var scriptUtilServices []byte

//go:embed scripts/util-packages.sh
var scriptUtilPackages []byte

//go:embed scripts/util-common.sh
var scriptUtilCommon []byte
