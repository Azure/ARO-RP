package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate ../../../hack/goruntool.sh mockgen -destination=../mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/$GOPACKAGE ClusterEnricher,BestEffortEnricher
//go:generate ../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../mocks/$GOPACKAGE/$GOPACKAGE.go
