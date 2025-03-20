package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type machineClientEnricher struct {
}

const (
	workerMachineSetsNamespace = "openshift-machine-api"
)

func (ce machineClientEnricher) Enrich(
	ctx context.Context,
	log *logrus.Entry,
	oc *api.OpenShiftCluster,
	k8scli kubernetes.Interface,
	configcli configclient.Interface,
	machinecli machineclient.Interface,
	operatorcli operatorclient.Interface,
) error {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return err
	}

	machinesets, err := machinecli.MachineV1beta1().MachineSets(workerMachineSetsNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	workerProfiles := make([]api.WorkerProfile, len(machinesets.Items))
	for i, machineset := range machinesets.Items {
		workerCount := 1
		if machineset.Spec.Replicas != nil {
			workerCount = int(*machineset.Spec.Replicas)
		}

		workerProfiles[i] = api.WorkerProfile{
			Name:  machineset.Name,
			Count: workerCount,
		}

		if machineset.Status.ReadyReplicas == 0 {
			log.Infof("no ready replicas in machine set %q", machineset.Name)
			continue
		}

		if machineset.Spec.Template.Spec.ProviderSpec.Value == nil {
			log.Infof("provider spec is missing in the machine set %q", machineset.Name)
			continue
		}

		machineProviderSpec := &machinev1beta1.AzureMachineProviderSpec{}
		err := json.Unmarshal(machineset.Spec.Template.Spec.ProviderSpec.Value.Raw, machineProviderSpec)
		if err != nil {
			log.Infof("failed to read provider spec from the machine set %q: %v", machineset.Name, err)
			continue
		}

		workerProfiles[i].VMSize = api.VMSize(machineProviderSpec.VMSize)
		workerProfiles[i].DiskSizeGB = int(machineProviderSpec.OSDisk.DiskSizeGB)
		workerProfiles[i].SubnetID = fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
			r.SubscriptionID, machineProviderSpec.NetworkResourceGroup, machineProviderSpec.Vnet, machineProviderSpec.Subnet,
		)

		encryptionAtHost := api.EncryptionAtHostDisabled
		if machineProviderSpec.SecurityProfile != nil &&
			machineProviderSpec.SecurityProfile.EncryptionAtHost != nil &&
			*machineProviderSpec.SecurityProfile.EncryptionAtHost {
			encryptionAtHost = api.EncryptionAtHostEnabled
		}

		workerProfiles[i].EncryptionAtHost = encryptionAtHost

		if machineProviderSpec.OSDisk.ManagedDisk.DiskEncryptionSet != nil {
			workerProfiles[i].DiskEncryptionSetID = machineProviderSpec.OSDisk.ManagedDisk.DiskEncryptionSet.ID
		}
	}

	sort.Slice(workerProfiles, func(i, j int) bool { return workerProfiles[i].Name < workerProfiles[j].Name })

	oc.Lock.Lock()
	defer oc.Lock.Unlock()

	oc.Properties.WorkerProfilesStatus = workerProfiles
	return nil
}

func (ce machineClientEnricher) SetDefaults(oc *api.OpenShiftCluster) {
	oc.Properties.WorkerProfilesStatus = nil
}
