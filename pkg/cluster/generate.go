package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/cluster/cluster.go github.com/Azure/ARO-RP/pkg/cluster Interface
//go:generate go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/samplesclient/versioned.go github.com/openshift/client-go/samples/clientset/versioned Interface
//go:generate go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/samples/samples.go github.com/openshift/client-go/samples/clientset/versioned/typed/samples/v1 SamplesV1Interface,ConfigInterface
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w -v ../util/mocks/cluster/cluster.go
