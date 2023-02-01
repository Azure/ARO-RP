package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"

	"github.com/Azure/go-autorest/autorest/azure"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

const (
	workerMachineSetsNamespace = "openshift-machine-api"
)

func newWorkerProfilesEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	maocli, err := machineclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &workerProfilesEnricherTask{
		log:    log,
		maocli: maocli,
		oc:     oc,
	}, nil
}

type workerProfilesEnricherTask struct {
	log    *logrus.Entry
	maocli machineclient.Interface
	oc     *api.OpenShiftCluster
}

func (ef *workerProfilesEnricherTask) FetchData(ctx context.Context, callbacks chan<- func(), errs chan<- error) {
	r, err := azure.ParseResourceID(ef.oc.ID)
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	machinesets, err := ef.maocli.MachineV1beta1().MachineSets(workerMachineSetsNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
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

		if machineset.Spec.Template.Spec.ProviderSpec.Value == nil {
			ef.log.Infof("provider spec is missing in the machine set %q", machineset.Name)
			continue
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(machineset.Spec.Template.Spec.ProviderSpec.Value.Raw, nil, nil)
		if err != nil {
			ef.log.Info(err)
			continue
		}
		machineProviderSpec, ok := obj.(*machinev1beta1.AzureMachineProviderSpec)
		if !ok {
			ef.log.Infof("failed to read provider spec from the machine set %q: %T", machineset.Name, obj)
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

	callbacks <- func() {
		ef.oc.Properties.WorkerProfiles = workerProfiles
	}
}

func (ef *workerProfilesEnricherTask) SetDefaults() {
	ef.oc.Properties.WorkerProfiles = nil
}
