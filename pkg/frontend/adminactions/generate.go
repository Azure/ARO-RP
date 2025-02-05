package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../util/mocks/$GOPACKAGE
//go:generate mockgen -source kubeactions.go -destination=../../util/mocks/$GOPACKAGE/kubeactions.go github.com/Azure/ARO-RP/pkg/frontend/$GOPACKAGE KubeActions
//go:generate mockgen -source azureactions.go -destination=../../util/mocks/$GOPACKAGE/azureactions.go github.com/Azure/ARO-RP/pkg/frontend/$GOPACKAGE AzureActions
//go:generate mockgen -source applens.go -destination=../../util/mocks/$GOPACKAGE/applens.go github.com/Azure/ARO-RP/pkg/frontend/$GOPACKAGE AppLensActions
