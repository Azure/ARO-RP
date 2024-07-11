package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../util/mocks/operator/$GOPACKAGE
//go:generate ../../../hack/goruntool.sh mockgen -destination=../../util/mocks/operator/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/operator/$GOPACKAGE Operator
//go:generate ../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../../util/mocks/operator/$GOPACKAGE/$GOPACKAGE.go
