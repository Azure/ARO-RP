package msidataplane

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../mocks/$GOPACKAGE/client_factory.go
//go:generate mockgen -destination=../mocks/$GOPACKAGE/client_factory.go github.com/Azure/msi-dataplane/pkg/dataplane ClientFactory
//go:generate goimports -local=github.com/Azure/ARO-RP -e -w ../mocks/$GOPACKAGE/client_factory.go

// mockgen unfortunately walks the type aliases in the dataplane package, and tries to import their
// underlying sources from an internal package, so the generated mock is useful as a first step but
// is not functional as-is
//  go:generate mockgen -destination=../mocks/$GOPACKAGE/client.go github.com/Azure/msi-dataplane/pkg/dataplane Client
//  go:generate goimports -local=github.com/Azure/ARO-RP -e -w ../mocks/$GOPACKAGE/client.go
