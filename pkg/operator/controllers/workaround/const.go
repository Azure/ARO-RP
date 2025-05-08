package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	// systemreserved workaround
	// Tweaked values from from https://github.com/openshift/kubernetes/blob/master/pkg/kubelet/apis/config/v1beta1/defaults_linux.go
	hardEviction                = "500Mi"
	nodeFsAvailable             = "10%"
	nodeFsInodes                = "5%"
	imageFs                     = "15%"
	labelName                   = "aro.openshift.io/limits"
	labelValue                  = ""
	kubeletConfigName           = "aro-limits"
	workerMachineConfigPoolName = "worker"
	memReserved                 = "2000Mi"
)
