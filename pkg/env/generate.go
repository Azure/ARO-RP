package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../util/mocks/$GOPACKAGE

// Need to use source mode as reflect mode always uses pkg "azcore/internal/exported"
//go:generate sh -c "for file in core env certificateRefresher; do go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/$GOPACKAGE/${DOLLAR}file.go -source ${DOLLAR}file.go -aux_files github.com/Azure/ARO-RP/pkg/env=core.go,github.com/Azure/ARO-RP/pkg/env=armhelper.go; done"

//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../util/mocks/$GOPACKAGE/$GOPACKAGE.go
//go:generate go run ../../vendor/github.com/alvaroloes/enumer -type Feature -output zz_generated_feature_enumer.go
