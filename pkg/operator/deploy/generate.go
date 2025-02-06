package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../util/mocks/operator/$GOPACKAGE
//go:generate mockgen -destination=../../util/mocks/operator/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/operator/$GOPACKAGE Operator
