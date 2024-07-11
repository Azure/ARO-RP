package azcore

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../../mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate ../../../../../hack/goruntool.sh mockgen -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/tokencredential.go -source=tokencredential.go
//go:generate ../../../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../../../mocks/azureclient/azuresdk/$GOPACKAGE/tokencredential.go
