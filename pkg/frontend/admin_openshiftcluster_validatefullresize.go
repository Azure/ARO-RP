package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	machineNamespace = "openshift-machine-api"
)

func (f *frontend) getValidateFullResize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	err := f._getValidateFullResize(log, ctx, r)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _getValidateFullResize(log *logrus.Entry, ctx context.Context, r *http.Request) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return err
	}
	kubeActions, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	azureActions, err := f.newStreamAzureAction(ctx, r, log)
	if err != nil {
		return err
	}

	ocMachines, err := getClusterMachines(log, ctx, kubeActions)
	if err != nil {
		return err
	}
	azureVMs, err := getAzureVMs(log, ctx, azureActions, doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	if err != nil {
		return err
	}

	err = validateClusterMachinesAndVMs(log, ocMachines, azureVMs)

	return err
}

type machineBasics struct {
	labelZone string
	specZone  string
	size      string
}

type azureVMBasics struct {
	status []string
	vmSize string
	zone   string
}

func getClusterMachines(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions) (map[string]machineBasics, error) {
	rawPods, err := kubeActions.KubeList(ctx, "Machine", machineNamespace)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	machines := &machinev1beta1.MachineList{}
	err = codec.NewDecoderBytes(rawPods, &codec.JsonHandle{}).Decode(machines)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode machines, %s", err.Error()))
	}

	var validationErrs []error
	filteredMachines := make(map[string]machineBasics)
	for _, machine := range machines.Items {
		if role, ok := machine.Labels["machine.openshift.io/cluster-api-machine-role"]; ok && role == "master" {
			providerSpec := &machinev1beta1.AzureMachineProviderSpec{}
			err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &providerSpec)
			if err != nil {
				return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode provider spec, %s", err.Error()))
			}
			filteredMachine := machineBasics{
				labelZone: machine.Labels["machine.openshift.io/zone"],
				specZone:  *providerSpec.Zone,
				size:      providerSpec.VMSize,
			}

			if filteredMachine.labelZone != filteredMachine.specZone {
				err := fmt.Errorf("machine %v has a mismatch between label zone %v and spec zone %v. These values should match", machine.Name, filteredMachine.labelZone, filteredMachine.specZone)
				log.Info(err)
				validationErrs = append(validationErrs, err)
				continue
			}
			filteredMachines[machine.Name] = filteredMachine
		}
	}

	if err := errors.Join(validationErrs...); err != nil {
		return nil, err
	}

	err = validateZoneDistribution(filteredMachines, func(m machineBasics) string { return m.specZone })
	if err != nil {
		return nil, err
	}
	return filteredMachines, nil
}

func getAzureVMs(log *logrus.Entry, ctx context.Context, azureAction adminactions.AzureActions, resGroupName string) (map[string]azureVMBasics, error) {
	masterVMs := make(map[string]azureVMBasics)
	clusterRGName := stringutils.LastTokenByte(resGroupName, '/')
	subscriptionObjects, err := azureAction.GroupResourceList(ctx)
	if err != nil {
		return nil, err
	}

	var validationErrs []error
	// To-Do: deal with high number of objects
	for _, object := range subscriptionObjects {
		vmStatuses := []string{}
		vmZones := []string{}
		apiVersion := azureclient.APIVersion(*object.Type)
		if apiVersion == "" {
			// If custom resource types, or any we don't have listed in pkg/util/azureclient/apiversions.go,
			// are returned, then skip over them instead of returning an error, otherwise it results in an
			// HTTP 500 and prevents the known resource types from being returned.
			// a.log.Warnf("API version not found for type %q", *res.Type)
			continue
		}
		if *object.Type == "Microsoft.Compute/virtualMachines" {
			if strings.Contains(*object.Name, "master-") {
				vm, err := azureAction.GetVirtualMachine(ctx, clusterRGName, *object.Name, mgmtcompute.InstanceView)
				if err != nil {
					return nil, err
				}

				for _, status := range *vm.InstanceView.Statuses {
					if status.Code == nil {
						continue
					}
					vmStatuses = append(vmStatuses, *status.Code)
				}

				if vm.Zones != nil {
					vmZones = *vm.Zones
				}

				if len(vmZones) == 0 {
					err := fmt.Errorf("azure VM %v has no availability zone configured", *object.Name)
					log.Info(err)
					validationErrs = append(validationErrs, err)
					continue
				}

				err = validateVMPowerState(log, vmStatuses, *object.Name)
				if err != nil {
					validationErrs = append(validationErrs, err)
				}

				masterVM := azureVMBasics{
					vmSize: string(vm.HardwareProfile.VMSize),
					status: vmStatuses,
					zone:   vmZones[0],
				}

				masterVMs[*object.Name] = masterVM
			}
		}
	}

	if err := errors.Join(validationErrs...); err != nil {
		return nil, err
	}

	for name, contents := range masterVMs {
		log.Warnf("Azure VM %v has vmSize %v, status %v and zone %v", name, contents.vmSize, contents.status, contents.zone)
	}

	err = validateZoneDistribution(masterVMs, func(m azureVMBasics) string { return m.zone })
	if err != nil {
		return nil, err
	}
	return masterVMs, nil
}

func validateClusterMachinesAndVMs(log *logrus.Entry, ocMachines map[string]machineBasics, azureVMs map[string]azureVMBasics) error {
	// assumptions: keys in both maps should match, azure VMs are named after Openshift VMs
	var validationErrs []error

	for name, machineSpec := range ocMachines {
		if _, ok := azureVMs[name]; !ok {
			err := fmt.Errorf("machine %v not found in Azure resources", name)
			log.Info(err)
			validationErrs = append(validationErrs, err)
			continue
		}

		if machineSpec.specZone != azureVMs[name].zone {
			err := fmt.Errorf("machine %v has zone %v in its spec, however Azure VM is running in zone %v", name, machineSpec.specZone, azureVMs[name].zone)
			log.Info(err)
			validationErrs = append(validationErrs, err)
		}

		if machineSpec.size != azureVMs[name].vmSize {
			err := fmt.Errorf("machine %v has size %v in its spec, however Azure VM is running a %v VM", name, machineSpec.size, azureVMs[name].vmSize)
			log.Info(err)
			validationErrs = append(validationErrs, err)
		}
	}

	return errors.Join(validationErrs...)
}

func validateZoneDistribution[T any](items map[string]T, getZone func(T) string) error {
	if len(items) != 3 {
		return fmt.Errorf("expected 3 items, got %d", len(items))
	}

	zones := make(map[string]bool, 3)
	for _, item := range items {
		zones[getZone(item)] = true
	}

	if len(zones) != 3 {
		return fmt.Errorf("items must be spread across 3 different zones, found %d zone(s)", len(zones))
	}

	return nil
}

func validateVMPowerState(log *logrus.Entry, vmStatuses []string, vmName string) error {
	if len(vmStatuses) != 2 { // We only expect 2 power states: ProvisioningState and PowerState ... if we have more or less statuses than that, the VM is not valid
		err := fmt.Errorf("expected 2 statuses for VM %s, but found %d: %s", vmName, len(vmStatuses), strings.Join(vmStatuses, ", "))
		log.Info(err)
		return err
	}

	var abnormalStatuses []string
	for _, status := range vmStatuses {
		if status != "ProvisioningState/succeeded" && status != "PowerState/running" {
			abnormalStatuses = append(abnormalStatuses, status)
		}
	}

	if len(abnormalStatuses) > 0 {
		err := fmt.Errorf("found unexpected statuses for VM %s: %s", vmName, strings.Join(abnormalStatuses, ", "))
		log.Info(err)
		return err
	}

	return nil
}
