package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "errors"

var fetchClusterVersionError = errors.New("error fetching ClusterVersion")
var fetchAROOperatorMasterDeploymentError = errors.New("error fetching ARO Operator master deployment")
var listAROOperatorDeploymentsError = errors.New("error listing ARO Operator deployments")
