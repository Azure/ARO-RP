package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/$GOPACKAGE ClusterEnricher,BestEffortEnricher
