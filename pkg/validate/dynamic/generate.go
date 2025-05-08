package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// XXX Using mockgen in source mode here to prevent mockgen from following
//     type alias azcore.TokenCredential to an internal azcore subpackage.
//     See https://go.uber.org/mock/issues/244

//go:generate rm -rf ../../../pkg/util/mocks/$GOPACKAGE
//go:generate mockgen -destination=../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go -source=dynamic.go
//go:generate mockgen -destination=../../../pkg/util/mocks/checkaccess/checkaccess.go github.com/Azure/checkaccess-v2-go-sdk/client RemotePDPClient
