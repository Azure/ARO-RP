package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"

	"github.com/Azure/go-autorest/autorest/azure"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	workerMachineSetsNamespace = "openshift-machine-api"
)

func newWorkerProfilesEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	client, err := maoclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &workerProfilesEnricherTask{
		log:    log,
		client: client,
		oc:     oc,
	}, nil
}

type workerProfilesEnricherTask struct {
	log    *logrus.Entry
	client maoclient.Interface
	oc     *api.OpenShiftCluster
}

func (ef *workerProfilesEnricherTask) FetchData(ctx context.Context, callbacks chan<- func(), errs chan<- error) {
	r, err := azure.ParseResourceID(ef.oc.ID)
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	machinesets, err := ef.client.MachineV1beta1().MachineSets(workerMachineSetsNamespace).List(ctx, metav1.ListOptions{})
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

		o, _, err := scheme.Codecs.UniversalDeserializer().Decode(machineset.Spec.Template.Spec.ProviderSpec.Value.Raw, nil, nil)
		if err != nil {
			ef.log.Info(err)
			continue
		}

		machineProviderSpec, ok := o.(*azureproviderv1beta1.AzureMachineProviderSpec)
		if !ok {
			// This should never happen: codecs uses scheme that has only one registered type
			// and if something is wrong with the provider spec - decoding should fail
			ef.log.Infof("failed to read provider spec from the machine set %q: %T", machineset.Name, o)
			continue
		}

		workerProfiles[i].VMSize = api.VMSize(machineProviderSpec.VMSize)
		workerProfiles[i].DiskSizeGB = int(machineProviderSpec.OSDisk.DiskSizeGB)
		workerProfiles[i].SubnetID = fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
			r.SubscriptionID, machineProviderSpec.NetworkResourceGroup, machineProviderSpec.Vnet, machineProviderSpec.Subnet,
		)
	}

	sort.Sort(byName(workerProfiles))

	callbacks <- func() {
		ef.oc.Properties.WorkerProfiles = workerProfiles
	}
}

func (ef *workerProfilesEnricherTask) SetDefaults() {
	ef.oc.Properties.WorkerProfiles = nil
}

// byName implements sort.Interface for []api.WorkerProfile based on the Name field.
type byName []api.WorkerProfile

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }
