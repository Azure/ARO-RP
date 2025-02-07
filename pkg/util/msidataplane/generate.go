package msidataplane

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../mocks/$GOPACKAGE/client_factory.go
//go:generate mockgen -destination=../mocks/$GOPACKAGE/client_factory.go github.com/Azure/msi-dataplane/pkg/dataplane ClientFactory

// mockgen unfortunately walks the type aliases in the dataplane package, and tries to import their
// underlying sources from an internal package, so the generated mock is useful as a first step but
// is not functional as-is, if ever fixed, add the compiler directive back
// mockgen -destination=../mocks/$GOPACKAGE/client.go github.com/Azure/msi-dataplane/pkg/dataplane Client
