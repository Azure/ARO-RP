package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate mockgen -destination=../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE ResourceSKUsClient
//go:generate mockgen -source ./capacityreservationgroups.go -destination=../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE/capacityreservationgroups.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE CapacityReservationGroupsClient
//go:generate mockgen -source ./capacityreservations.go -destination=../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE/capacityreservations.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE CapacityReservationsClient
//go:generate mockgen -source ./virtualmachines.go -destination=../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE/virtualmachines.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE VirtualMachinesClient
