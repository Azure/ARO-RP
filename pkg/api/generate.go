package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../vendor/github.com/alvaroloes/enumer -type InstallPhase -output zz_generated_installphase_enumer.go
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../util/mocks/api/api.go
