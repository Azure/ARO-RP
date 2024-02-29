package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../vendor/github.com/jewzaam/go-cosmosdb/cmd/gencosmosdb github.com/Azure/ARO-RP/pkg/api,AsyncOperationDocument github.com/Azure/ARO-RP/pkg/api,BillingDocument github.com/Azure/ARO-RP/pkg/api,GatewayDocument github.com/Azure/ARO-RP/pkg/api,MonitorDocument github.com/Azure/ARO-RP/pkg/api,OpenShiftClusterDocument github.com/Azure/ARO-RP/pkg/api,SubscriptionDocument github.com/Azure/ARO-RP/pkg/api,OpenShiftVersionDocument github.com/Azure/ARO-RP/pkg/api,ClusterManagerConfigurationDocument github.com/Azure/ARO-RP/pkg/api,PlatformWorkloadIdentityRoleSetDocument github.com/Azure/ARO-RP/pkg/api,MaintenanceManifestDocument
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ./
//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../../util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/database/$GOPACKAGE PermissionClient
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../util/mocks/$GOPACKAGE/$GOPACKAGE.go
