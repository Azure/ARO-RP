package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../mocks/$GOPACKAGE
//go:generate mockgen -destination=../mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/$GOPACKAGE Manager
//go:generate goimports -local=github.com/Azure/ARO-RP -e -w ../mocks/$GOPACKAGE/$GOPACKAGE.go
