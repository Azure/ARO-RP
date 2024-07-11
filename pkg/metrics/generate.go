package metrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../util/mocks/$GOPACKAGE
//go:generate ../../hack/goruntool.sh mockgen -destination=../util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/$GOPACKAGE Emitter
//go:generate ../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../util/mocks/$GOPACKAGE/$GOPACKAGE.go
