package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// XXX Using mockgen in source mode here to prevent mockgen from following
//     type alias azcore.TokenCredential to an internal azcore subpackage.
//     See https://github.com/golang/mock/issues/244

//go:generate rm -rf ../../../../pkg/util/mocks/$GOPACKAGE
//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go -source=dynamic.go
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go
