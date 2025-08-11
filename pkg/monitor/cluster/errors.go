package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "errors"

var errFetchClusterVersion = errors.New("error fetching ClusterVersion")
var errFetchAROOperatorMasterDeployment = errors.New("error fetching ARO Operator master deployment")
var errListAROOperatorDeployments = errors.New("error listing ARO Operator deployments")
