package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../util/mocks/operator/controllers/$GOPACKAGE
//go:generate ../../../../hack/goruntool.sh mockgen -destination=../../../util/mocks/operator/controllers/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/operator/controllers/$GOPACKAGE Workaround
//go:generate ../../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../../../util/mocks/operator/controllers/$GOPACKAGE/$GOPACKAGE.go
