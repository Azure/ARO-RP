package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var kindStartOrder = map[string]int{
	"CustomResourceDefinition":   1,
	"Namespace":                  2,
	"SecurityContextConstraints": 3,
	"ClusterRole":                4,
	"ClusterRoleBinding":         5,
	"ServiceAccount":             6,
	"Secret":                     7,
	"ConfigMap":                  8,
	"StorageClass":               9,
	"Service":                    10,
	"Deployment":                 11,
	"DaemonSet":                  12,
	"CronJob":                    13,
	"StatefulSet":                14,
	"Cluster":                    15,
}

// KindLess is to be used in a sort.Slice() comparison. It is to help make
// sure that resources are created in an order that causes a reliable startup.
func KindLess(i, j string) bool {
	io, ok := kindStartOrder[i]
	if !ok {
		io = 10
	}
	jo, ok := kindStartOrder[j]
	if !ok {
		jo = 10
	}
	return io < jo
}
