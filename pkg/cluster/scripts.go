package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import _ "embed"

//go:embed scripts/fixetcd.sh
var backupOrFixEtcd string
