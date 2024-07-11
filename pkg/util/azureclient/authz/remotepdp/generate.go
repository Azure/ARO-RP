package remotepdp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../../mocks/azureclient/authz/$GOPACKAGE
//go:generate ../../../../../hack/goruntool.sh mockgen -destination=../../../mocks/azureclient/authz/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/authz/$GOPACKAGE RemotePDPClient
//go:generate ../../../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../../../mocks/azureclient/authz/$GOPACKAGE/$GOPACKAGE.go
