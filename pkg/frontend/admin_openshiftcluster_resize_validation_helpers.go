package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	corev1 "k8s.io/api/core/v1"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/azurezones"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type zoneTopology string

const (
	controlPlaneReplicaCount = azurezones.CONTROL_PLANE_MACHINE_COUNT

	machineNamespace           = "openshift-machine-api"
	machineLabelClusterAPIRole = "machine.openshift.io/cluster-api-machine-role"
	machineLabelZone           = "machine.openshift.io/zone"
	machineLabelInstanceType   = "machine.openshift.io/instance-type"
	machineRoleMaster          = "master"

	nodeLabelInstanceType     = "node.kubernetes.io/instance-type"
	nodeLabelBetaInstanceType = "beta.kubernetes.io/instance-type"

	// SRE-facing label for the raw empty-zone sentinel; not a zoneTopology value.
	zoneDisplayRegionalNoZone = "regional/no-zone"

	zoneTopologyRegional  zoneTopology = "regional"
	zoneTopologyThreeZone zoneTopology = "three-zone"
)

type machineValidationData struct {
	labelZone         string
	specZone          string
	size              string
	phase             string
	labelInstanceType string
}

type azureVMValidationData struct {
	status []string
	vmSize string
	zone   string
}

type nodeValidationData struct {
	nodeInstanceType string
	betaInstanceType string
}

func describeZone(zone string) string {
	if zone == "" {
		return zoneDisplayRegionalNoZone
	}
	return zone
}

func describeZones(zones []string) []string {
	displayZones := make([]string, 0, len(zones))
	for _, zone := range zones {
		displayZones = append(displayZones, describeZone(zone))
	}
	return displayZones
}

func unmarshalAzureMachineProviderSpec(machine *machinev1beta1.Machine) (*machinev1beta1.AzureMachineProviderSpec, error) {
	if machine.Spec.ProviderSpec.Value == nil {
		return nil, fmt.Errorf("provider spec value is nil")
	}

	providerSpec := &machinev1beta1.AzureMachineProviderSpec{}
	if err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, providerSpec); err != nil {
		return nil, err
	}

	return providerSpec, nil
}

func getClusterMachines(ctx context.Context, kubeActions adminactions.KubeActions) (map[string]machineValidationData, error) {
	machines := make(map[string]machineValidationData)

	rawMachines, err := kubeActions.KubeList(ctx, "Machine.machine.openshift.io", machineNamespace)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError,
			"controlPlaneMachines",
			err.Error(),
		)
	}

	machineList := &machinev1beta1.MachineList{}
	err = codec.NewDecoderBytes(rawMachines, &codec.JsonHandle{}).Decode(machineList)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError,
			"controlPlaneMachines",
			fmt.Sprintf("failed to decode machines, %s", err.Error()),
		)
	}

	for _, machine := range machineList.Items {
		if role, ok := machine.Labels[machineLabelClusterAPIRole]; ok && role == machineRoleMaster {
			providerSpec, err := unmarshalAzureMachineProviderSpec(&machine)
			if err != nil {
				return nil, api.NewCloudError(
					http.StatusInternalServerError,
					api.CloudErrorCodeInternalServerError,
					fmt.Sprintf("controlPlaneMachine/%s", machine.Name),
					fmt.Sprintf("failed to parse provider spec for machine %s: %v", machine.Name, err),
				)
			}

			phase := ""
			if machine.Status.Phase != nil {
				phase = *machine.Status.Phase
			}

			machineBasic := machineValidationData{
				labelZone:         machine.Labels[machineLabelZone],
				specZone:          providerSpec.Zone,
				size:              providerSpec.VMSize,
				phase:             phase,
				labelInstanceType: machine.Labels[machineLabelInstanceType],
			}

			machines[machine.Name] = machineBasic
		}
	}

	return machines, nil
}

func validateClusterMachines(log *logrus.Entry, machines map[string]machineValidationData) (map[string]machineValidationData, error) {
	if len(machines) != controlPlaneReplicaCount {
		return nil, fmt.Errorf("expected %d machines, got %d", controlPlaneReplicaCount, len(machines))
	}

	var validationErrs []error
	filteredMachines := make(map[string]machineValidationData)

	for name, machine := range machines {
		if machine.phase != "Running" {
			phase := "nil"
			if machine.phase != "" {
				phase = machine.phase
			}
			err := fmt.Errorf("machine %s status phase is not Running, current phase is %s", name, phase)
			log.Info(err)
			validationErrs = append(validationErrs, err)
			continue
		}

		if machine.labelZone != machine.specZone {
			err := fmt.Errorf("machine %s has a mismatch between label zone %s and spec zone %s. These values should match", name, describeZone(machine.labelZone), describeZone(machine.specZone))
			log.Info(err)
			validationErrs = append(validationErrs, err)
			continue
		}

		if machine.labelInstanceType == "" || machine.labelInstanceType != machine.size {
			labelValue := machine.labelInstanceType
			if labelValue == "" {
				labelValue = "<missing>"
			}
			err := fmt.Errorf("machine %s has a mismatch between label instance-type %s and instance type defined in the spec %s. These values should match", name, labelValue, machine.size)
			log.Info(err)
			validationErrs = append(validationErrs, err)
			continue
		}

		filteredMachines[name] = machine
	}

	sizes := make(map[string][]string)
	for name, m := range filteredMachines {
		sizes[m.size] = append(sizes[m.size], name)
	}
	// During a partial resize, old and new VM sizes can coexist temporarily, so warn instead of failing.
	if len(sizes) > 1 {
		log.Warnf("different control plane VM sizes detected (may indicate a partial resize): %v", sizes)
	}

	if err := errors.Join(validationErrs...); err != nil {
		return nil, err
	}

	_, err := classifyZoneTopology(filteredMachines, func(m machineValidationData) string { return m.specZone })
	if err != nil {
		return nil, err
	}
	return filteredMachines, nil
}

func getAzureVMs(log *logrus.Entry, ctx context.Context, azureAction adminactions.AzureActions, resGroupName string, machines map[string]machineValidationData) (map[string]azureVMValidationData, error) {
	masterVMs := make(map[string]azureVMValidationData)
	clusterRGName := stringutils.LastTokenByte(resGroupName, '/')

	var validationErrs []error
	for machineName := range machines {
		vmStatuses := []string{}
		vmZones := []string{}

		vm, err := azureAction.GetVirtualMachine(ctx, clusterRGName, machineName, mgmtcompute.InstanceView)
		if err != nil {
			// A VM lookup failure means we could not inspect that machine at all, so
			// fail fast instead of returning a partial validation result. In contrast,
			// power-state mismatches are aggregated below because the VM was fetched
			// successfully and we can still report all unhealthy instances together.
			return nil, api.NewCloudError(
				http.StatusInternalServerError,
				api.CloudErrorCodeInternalServerError,
				fmt.Sprintf("controlPlaneVM/%s", machineName),
				fmt.Sprintf("failed to get Azure VM %s: %v", machineName, err),
			)
		}

		if vm.InstanceView != nil && vm.InstanceView.Statuses != nil {
			for _, status := range *vm.InstanceView.Statuses {
				if status.Code == nil {
					continue
				}
				vmStatuses = append(vmStatuses, *status.Code)
			}
		}

		if vm.Zones != nil {
			vmZones = *vm.Zones
		}

		err = validateVMPowerState(log, vmStatuses, machineName)
		if err != nil {
			validationErrs = append(validationErrs, err)
		}

		vmSize := ""
		if vm.HardwareProfile != nil {
			vmSize = string(vm.HardwareProfile.VMSize)
		}

		zone := ""
		if len(vmZones) > 0 {
			zone = vmZones[0]
		}
		// An absent Azure zone intentionally represents a regional VM.

		masterVM := azureVMValidationData{
			vmSize: vmSize,
			status: vmStatuses,
			zone:   zone,
		}

		masterVMs[machineName] = masterVM
	}

	if err := errors.Join(validationErrs...); err != nil {
		return nil, err
	}

	_, err := classifyZoneTopology(masterVMs, func(m azureVMValidationData) string { return m.zone })
	if err != nil {
		return nil, err
	}
	return masterVMs, nil
}

func validateClusterMachinesAndVMs(log *logrus.Entry, ocMachines map[string]machineValidationData, azureVMs map[string]azureVMValidationData) error {
	// assumptions: keys in both maps should match, azure VMs are named after Openshift VMs
	var validationErrs []error

	for name, machineSpec := range ocMachines {
		if _, ok := azureVMs[name]; !ok {
			err := fmt.Errorf("machine %s not found in Azure resources", name)
			log.Info(err)
			validationErrs = append(validationErrs, err)
			continue
		}

		if machineSpec.specZone != azureVMs[name].zone {
			err := fmt.Errorf("machine %s has zone %s in its spec, however Azure VM is running in zone %s", name, describeZone(machineSpec.specZone), describeZone(azureVMs[name].zone))
			log.Info(err)
			validationErrs = append(validationErrs, err)
		}

		if machineSpec.size != azureVMs[name].vmSize {
			err := fmt.Errorf("machine %s has size %s in its spec, however Azure VM is running a %s VM", name, machineSpec.size, azureVMs[name].vmSize)
			log.Info(err)
			validationErrs = append(validationErrs, err)
		}
	}

	return errors.Join(validationErrs...)
}

func validateClusterMachinesAndNodes(log *logrus.Entry, ocMachines map[string]machineValidationData, ocNodes map[string]nodeValidationData) error {
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

// classifyZoneTopology validates the recognized control-plane placement shapes.
// It is generic so the same rules can be applied to Machines, VMs, and capacity
// reservations without translating them into a resize-specific data structure.
func classifyZoneTopology[T any](items map[string]T, getZone func(T) string) (zoneTopology, error) {
	if len(items) != controlPlaneReplicaCount {
		return "", fmt.Errorf("expected %d items, got %d", controlPlaneReplicaCount, len(items))
	}

	zones := make(map[string]bool, controlPlaneReplicaCount)
	var soleZone string
	for _, item := range items {
		soleZone = getZone(item)
		zones[soleZone] = true
	}

	if len(zones) == 1 {
		// One empty zone is regional; one explicit zone is a single-zone layout,
		// which resizecontrolplane must reject explicitly instead of treating as
		// a mixed or regional topology.
		if soleZone == "" {
			return zoneTopologyRegional, nil
		}
		return "", fmt.Errorf("single-zone control plane topology is unsupported for resize validation: zones [%q]", soleZone)
	}

	if len(zones) == controlPlaneReplicaCount && !zones[""] {
		// Three-zone placement requires every replica to have a distinct explicit zone.
		return zoneTopologyThreeZone, nil
	}

	zoneNames := make([]string, 0, len(zones))
	for zone := range zones {
		zoneNames = append(zoneNames, zone)
	}
	// Sort zones to keep validation errors deterministic across map iterations.
	sort.Strings(zoneNames)
	return "", fmt.Errorf("items have unsupported mixed zone topology: zones %q", describeZones(zoneNames))
}

func validateVMPowerState(log *logrus.Entry, vmStatuses []string, vmName string) error {
	// We require at least the provisioning and power states; any additional
	// statuses are validated below and rejected if unexpected.
	if len(vmStatuses) < 2 {
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

func validateClusterNodes(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions) (map[string]nodeValidationData, error) {
	var validationErrs []error
	rawNodes, err := kubeActions.KubeList(ctx, "Node", "")
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError,
			"controlPlaneNodes",
			err.Error(),
		)
	}

	nodeList := &corev1.NodeList{}
	err = codec.NewDecoderBytes(rawNodes, &codec.JsonHandle{}).Decode(nodeList)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError,
			"controlPlaneNodes",
			fmt.Sprintf("failed to decode nodes, %s", err.Error()),
		)
	}

	controlPlaneNodesFound := make(map[string]nodeValidationData)
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

			nodeInfo := nodeValidationData{
				nodeInstanceType: node.Labels[nodeLabelInstanceType],
				betaInstanceType: node.Labels[nodeLabelBetaInstanceType],
			}
			controlPlaneNodesFound[node.Name] = nodeInfo

			if nodeInfo.betaInstanceType != nodeInfo.nodeInstanceType {
				err := fmt.Errorf("node %s has a mismatch between labels. %s: %s %s: %s", node.Name, nodeLabelInstanceType, nodeInfo.nodeInstanceType, nodeLabelBetaInstanceType, nodeInfo.betaInstanceType)
				log.Info(err)
				validationErrs = append(validationErrs, err)
			}
		}
	}

	if len(controlPlaneNodesFound) != controlPlaneReplicaCount {
		nodeNames := make([]string, 0, len(controlPlaneNodesFound))
		for name := range controlPlaneNodesFound {
			nodeNames = append(nodeNames, name)
		}
		// Sort node names so count-mismatch errors stay deterministic.
		sort.Strings(nodeNames)
		err := fmt.Errorf("expected %d control plane nodes, found %d: [%s]", controlPlaneReplicaCount, len(controlPlaneNodesFound), strings.Join(nodeNames, ", "))
		log.Info(err)
		validationErrs = append(validationErrs, err)
	}

	if err := errors.Join(validationErrs...); err != nil {
		return nil, err
	}

	return controlPlaneNodesFound, nil
}
