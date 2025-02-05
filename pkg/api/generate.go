package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate enumer -type InstallPhase -output zz_generated_installphase_enumer.go
//go:generate mockgen -destination=../util/mocks/api/api.go github.com/Azure/ARO-RP/pkg/api SyncSetConverter,MachinePoolConverter,SyncIdentityProviderConverter,SecretConverter
