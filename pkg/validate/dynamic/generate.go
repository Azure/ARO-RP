package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// XXX Using mockgen in source mode here to prevent mockgen from following
//     type alias azcore.TokenCredential to an internal azcore subpackage.
//     See https://github.com/golang/mock/issues/244

//go:generate rm -rf ../../../pkg/util/mocks/$GOPACKAGE
//go:generate ../../../hack/goruntool.sh mockgen -destination=../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go -source=dynamic.go
//go:generate ../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go
