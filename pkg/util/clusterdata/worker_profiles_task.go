package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"sort"

	"github.com/Azure/go-autorest/autorest/azure"
	azureproviderv1beta1 "github.com/openshift/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
	clusterapi "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	workerMachineSetsNamespace = "openshift-machine-api"
)

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	err := azureproviderv1beta1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
}

func newWorkerProfilesEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	client, err := clusterapi.NewForConfig(restConfig)
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
	client clusterapi.Interface
	oc     *api.OpenShiftCluster
}

func (ef *workerProfilesEnricherTask) FetchData(callbacks chan<- func(), errs chan<- error) {
	r, err := azure.ParseResourceID(ef.oc.ID)
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	machinesets, err := ef.client.MachineV1beta1().MachineSets(workerMachineSetsNamespace).List(metav1.ListOptions{})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	workerProfiles := make([]api.WorkerProfile, len(machinesets.Items), len(machinesets.Items))
	for i, machineset := range machinesets.Items {
		o, _, err := codecs.UniversalDeserializer().Decode(machineset.Spec.Template.Spec.ProviderSpec.Value.Raw, nil, nil)
		if err != nil {
			ef.log.Error(err)
			errs <- err
			return
		}

		machineProviderSpec, ok := o.(*azureproviderv1beta1.AzureMachineProviderSpec)
		if !ok {
			ef.log.Errorf("failed to read provider spec from the machine set %q: %T", machineset.Name, o)
			errs <- err
			return
		}

		workerCount := 1
		if machineset.Spec.Replicas != nil {
			workerCount = int(*machineset.Spec.Replicas)
		}

		workerProfiles[i] = api.WorkerProfile{
			Name:       machineset.Name,
			VMSize:     api.VMSize(machineProviderSpec.VMSize),
			DiskSizeGB: int(machineProviderSpec.OSDisk.DiskSizeGB),
			SubnetID: fmt.Sprintf(
				"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
				r.SubscriptionID, machineProviderSpec.NetworkResourceGroup, machineProviderSpec.Vnet, machineProviderSpec.Subnet,
			),
			Count: workerCount,
		}
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
