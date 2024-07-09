package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import _ "embed"

//go:embed scripts/aro-coredns.conf.gotmpl
var configFile string

//go:embed scripts/aro-coredns.service.gotmpl
var unitFile string

//go:embed scripts/aro-coredns-pre.sh.gotmpl
var preScriptFile string

//go:embed scripts/99-aro-coredns-restart.gotmpl
var restartScript string
