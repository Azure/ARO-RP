package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../util/mocks/$GOPACKAGE
//go:generate go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/$GOPACKAGE ClusterManager
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../util/mocks/$GOPACKAGE/$GOPACKAGE.go
//go:generate go run ../../vendor/github.com/golang/mock/mockgen -build_flags=--mod=mod -destination=../util/mocks/hive/clientset/versioned/typed/hive/v1/hivev1.go github.com/openshift/hive/pkg/client/clientset/versioned/typed/hive/v1 ClusterDeploymentInterface,ClusterDeploymentsGetter,HiveV1Interface
//go:generate go run ../../vendor/github.com/golang/mock/mockgen -build_flags=--mod=mod -destination=../util/mocks/hive/clientset/versioned/interface.go github.com/openshift/hive/pkg/client/clientset/versioned Interface
