package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../util/mocks/$GOPACKAGE
//go:generate mockgen -source quota_validation.go -destination=../util/mocks/$GOPACKAGE/quota_validation.go github.com/Azure/ARO-RP/pkg/frontend QuotaValidator
//go:generate mockgen -source providers_validation.go -destination=../util/mocks/$GOPACKAGE/providers_validation.go github.com/Azure/ARO-RP/pkg/frontend ProvidersValidator
//go:generate mockgen -source sku_validation.go -destination=../util/mocks/$GOPACKAGE/sku_validation.go github.com/Azure/ARO-RP/pkg/frontend SkuValidator
//go:generate mockgen -source adminreplies.go -destination=../util/mocks/$GOPACKAGE/adminreplies.go github.com/Azure/ARO-RP/pkg/frontend StreamResponder
