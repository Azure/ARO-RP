package azmetrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../../mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate mockgen -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/metrics.go -source=metrics.go
