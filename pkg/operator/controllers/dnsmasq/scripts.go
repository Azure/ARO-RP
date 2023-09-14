package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import _ "embed"

//go:embed scripts/dnsmasq.conf.gotmpl
var configFile string

//go:embed scripts/dnsmasq.service.gotmpl
var unitFile string

//go:embed scripts/aro-dnsmasq-pre.sh.gotmpl
var preScriptFile string

//go:embed scripts/99-dnsmasq-restart.gotmpl
var restartScript string
