package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate ../../hack/goruntool.sh mockgen -destination=../util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/$GOPACKAGE Interface
//go:generate ../../hack/goruntool.sh mockgen -destination=../util/mocks/samplesclient/versioned.go github.com/openshift/client-go/samples/clientset/versioned Interface
//go:generate ../../hack/goruntool.sh mockgen -destination=../util/mocks/samples/samples.go github.com/openshift/client-go/samples/clientset/versioned/typed/samples/v1 SamplesV1Interface,ConfigInterface
//go:generate ../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../util/mocks/$GOPACKAGE/$GOPACKAGE.go
