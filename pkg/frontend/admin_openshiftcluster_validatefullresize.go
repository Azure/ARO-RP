package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

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

	corev1 "k8s.io/api/core/v1"

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

// convertErrorLineEndings converts newlines to a clearer separator " | ", as it seems that the new lines are not being parsed in GA
// (or we need to do deeper changes to have nicer error messages)
func convertErrorLineEndings(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	errMsg = strings.ReplaceAll(errMsg, "\n", " | ")
	return fmt.Errorf("%s", errMsg)
}

func (f *frontend) getPostResizeControlPlaneVMs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		apiErr := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		adminReply(log, w, nil, nil, apiErr)
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		apiErr := api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
		adminReply(log, w, nil, nil, apiErr)
		return
	case err != nil:
		adminReply(log, w, nil, nil, err)
		return
	}
	kubeActions, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		apiErr := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		adminReply(log, w, nil, nil, apiErr)
		return
	}

	azureActions, err := f.newStreamAzureAction(ctx, r, log)
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}
	err = f._getPostResizeControlPlaneVMs(log, ctx, kubeActions, azureActions, doc)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _getPostResizeControlPlaneVMs(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions, azureActions adminactions.AzureActions, doc *api.OpenShiftClusterDocument) error {
	ocMachines, err := getClusterMachines(log, ctx, kubeActions)
	if err != nil {
		return convertErrorLineEndings(err)
	}
	azureVMs, err := getAzureVMs(log, ctx, azureActions, doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	if err != nil {
		return convertErrorLineEndings(err)
	}

	err = validateClusterMachinesAndVMs(log, ocMachines, azureVMs)
	if err != nil {
		return convertErrorLineEndings(err)
	}

	ocNodes, err := validateClusterNodes(log, ctx, kubeActions)
	if err != nil {
		return convertErrorLineEndings(err)
	}

	err = validateClusterMachinesAndNodes(log, ocMachines, ocNodes)
	return convertErrorLineEndings(err)
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

type nodeBasics struct {
	nodeInstanceType string
	betaInstanceType string
}

func getClusterMachines(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions) (map[string]machineBasics, error) {
	var validationErrs []error
	filteredMachines := make(map[string]machineBasics)
	foundMachineSize := ""

	rawPods, err := kubeActions.KubeList(ctx, "Machine", machineNamespace)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	machines := &machinev1beta1.MachineList{}
	err = codec.NewDecoderBytes(rawPods, &codec.JsonHandle{}).Decode(machines)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode machines, %s", err.Error()))
	}

	for _, machine := range machines.Items {
		if role, ok := machine.Labels["machine.openshift.io/cluster-api-machine-role"]; ok && role == "master" {
			providerSpec := &machinev1beta1.AzureMachineProviderSpec{}
			err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &providerSpec)
			if err != nil {
				return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode provider spec, %s", err.Error()))
			}

			if machine.Status.Phase == nil || *machine.Status.Phase != "Running" {
				phase := "nil"
				if machine.Status.Phase != nil {
					phase = *machine.Status.Phase
				}
				err := fmt.Errorf("machine %s status phase is not Running, current phase is %s", machine.Name, phase)
				log.Info(err)
				validationErrs = append(validationErrs, err)
				continue
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

			machineLabelSize, ok := machine.Labels["machine.openshift.io/instance-type"]
			if !ok || machineLabelSize != filteredMachine.size {
				labelValue := machineLabelSize
				if !ok {
					labelValue = "<missing>"
				}
				err := fmt.Errorf("machine %s has a mismatch between label instance-type %s and instance type defined in the spec %s. These values should match", machine.Name, labelValue, filteredMachine.size)
				log.Info(err)
				validationErrs = append(validationErrs, err)
				continue
			}

			if foundMachineSize == "" {
				foundMachineSize = filteredMachine.size // we'll keep the machine size of the first machine to compare it with the rest
			}

			if filteredMachine.size != foundMachineSize {
				err := fmt.Errorf("machine %s has size %s, however previous machines had %s. All machines should have the same size", machine.Name, filteredMachine.size, foundMachineSize)
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

func validateClusterMachinesAndNodes(log *logrus.Entry, ocMachines map[string]machineBasics, ocNodes map[string]nodeBasics) error {
	// assumptions: keys in both maps should match, nodes are named after machines
	var validationErrs []error

	for name, machineSpec := range ocMachines {
		if _, ok := ocNodes[name]; !ok {
			err := fmt.Errorf("machine %s not found in cluster nodes", name)
			log.Info(err)
			validationErrs = append(validationErrs, err)
			continue
		}

		if machineSpec.size != ocNodes[name].nodeInstanceType {
			err := fmt.Errorf("machine %s has size %s in its spec, however node has instance-type %s", name, machineSpec.size, ocNodes[name].nodeInstanceType)
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

func validateClusterNodes(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions) (map[string]nodeBasics, error) {
	var validationErrs []error
	rawNodes, err := kubeActions.KubeList(ctx, "Node", "")
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	nodeList := &corev1.NodeList{}
	err = codec.NewDecoderBytes(rawNodes, &codec.JsonHandle{}).Decode(nodeList)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode nodes, %s", err.Error()))
	}

	controlPlaneNodesFound := make(map[string]nodeBasics)
	for _, node := range nodeList.Items {
		if role, ok := node.Labels["node-role.kubernetes.io/master"]; ok && role == "" {
			if node.Spec.Unschedulable {
				err := fmt.Errorf("node %s is unschedulable", node.Name)
				log.Info(err)
				validationErrs = append(validationErrs, err)
			}

			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
					err := fmt.Errorf("node %s is not ready", node.Name)
					log.Info(err)
					validationErrs = append(validationErrs, err)
				}
			}

			nodeInfo := nodeBasics{
				nodeInstanceType: node.Labels["node.kubernetes.io/instance-type"],
				betaInstanceType: node.Labels["beta.kubernetes.io/instance-type"],
			}
			controlPlaneNodesFound[node.Name] = nodeInfo

			if nodeInfo.betaInstanceType != nodeInfo.nodeInstanceType {
				err := fmt.Errorf("node %s has a mismatch between labels. node.kubernetes.io/instance-type: %s beta.kubernetes.io/instance-type: %s", node.Name, nodeInfo.nodeInstanceType, nodeInfo.betaInstanceType)
				log.Info(err)
				validationErrs = append(validationErrs, err)
			}
		}
	}

	if len(controlPlaneNodesFound) != 3 {
		nodeNames := make([]string, 0, len(controlPlaneNodesFound))
		for name := range controlPlaneNodesFound {
			nodeNames = append(nodeNames, name)
		}
		err := fmt.Errorf("expected 3 control plane nodes, found %d: [%s]", len(controlPlaneNodesFound), strings.Join(nodeNames, ", "))
		log.Info(err)
		validationErrs = append(validationErrs, err)
	}

	if err := errors.Join(validationErrs...); err != nil {
		return nil, err
	}

	return controlPlaneNodesFound, nil
}
