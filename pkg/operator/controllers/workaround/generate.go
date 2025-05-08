package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../util/mocks/operator/controllers/$GOPACKAGE
//go:generate mockgen -destination=../../../util/mocks/operator/controllers/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/operator/controllers/$GOPACKAGE Workaround
