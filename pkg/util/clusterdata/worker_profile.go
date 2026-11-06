package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/go-autorest/autorest/azure"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type machineClientEnricher struct{}

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

		// A MachineSet with no ready replicas is unhealthy, but its desired ProviderSpec
		// (and therefore its subnet) is still valid and must be reported. Bailing out here
		// used to leave WorkerProfilesStatus with an empty SubnetID, which downstream
		// consumers (notably the storage-account firewall rebuild in migrateStorageAccounts)
		// would then drop, revoking the worker's storage/image-registry access once it
		// recovered. Keep going so the configured subnet is always populated; only a
		// genuinely unreadable ProviderSpec (handled below) leaves SubnetID empty.
		if machineset.Status.ReadyReplicas == 0 {
			log.Infof("no ready replicas in machine set %q; still reporting its configured subnet", machineset.Name)
		}

		if machineset.Spec.Template.Spec.ProviderSpec.Value == nil {
			log.Infof("provider spec is missing in the machine set %q", machineset.Name)
			continue
		}

		machineProviderSpec, err := safeUnmarshalProviderSpec(machineset.Spec.Template.Spec.ProviderSpec.Value.Raw)
		if err != nil {
			log.Infof("failed to read provider spec from the machine set %q: %v", machineset.Name, err)
			continue
		}

		workerProfiles[i].VMSize = api.VMSize(machineProviderSpec.VMSize)
		workerProfiles[i].DiskSizeGB = int(machineProviderSpec.OSDisk.DiskSizeGB)
		// Only assemble the SubnetID when every network field is present. A partial spec
		// would otherwise yield a non-empty but malformed ID that slips past the empty-
		// SubnetID guards in downstream consumers and hard-fails on parse/Get.
		if machineProviderSpec.NetworkResourceGroup != "" && machineProviderSpec.Vnet != "" && machineProviderSpec.Subnet != "" {
			workerProfiles[i].SubnetID = fmt.Sprintf(
				"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
				r.SubscriptionID, machineProviderSpec.NetworkResourceGroup, machineProviderSpec.Vnet, machineProviderSpec.Subnet,
			)
		}

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

// Azure availability zones are expected to be strings despite being numeric.
// YAML conversion can cause these numeric values to be parsed as ints unless
// they are explicitly wrapped with "". We forcibly convert the zone field
// to a string on the unstructured object here before converting it to the
// typed spec instance.
func safeUnmarshalProviderSpec(raw []byte) (*machinev1beta1.AzureMachineProviderSpec, error) {
	u := unstructured.Unstructured{}
	if err := u.UnmarshalJSON(raw); err != nil {
		return nil, err
	}
	zoneRaw, hasZone, err := unstructured.NestedFieldNoCopy(u.Object, "zone")
	if err != nil {
		return nil, err
	}
	if hasZone {
		if err := unstructured.SetNestedField(u.Object, fmt.Sprintf("%v", zoneRaw), "zone"); err != nil {
			return nil, err
		}
	}

	tagsRaw, hasTags, err := unstructured.NestedMap(u.Object, "tags")
	if err != nil {
		return nil, err
	}
	if hasTags {
		tagsAsString := map[string]any{}
		for k, v := range tagsRaw {
			tagsAsString[k] = fmt.Sprintf("%v", v)
		}
		if err := unstructured.SetNestedMap(u.Object, tagsAsString, "tags"); err != nil {
			return nil, err
		}
	}

	providerSpec := &machinev1beta1.AzureMachineProviderSpec{}
	err = kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, providerSpec)

	return providerSpec, err
}
