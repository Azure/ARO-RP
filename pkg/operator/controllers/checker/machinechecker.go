package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	azureproviderv1beta1 "github.com/openshift/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	clusterapi "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset"
	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/api"
	aro "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	utilmachine "github.com/Azure/ARO-RP/pkg/util/machine"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

const (
	machineSetsNamespace = "openshift-machine-api"
)

// MachineChecker reconciles the alertmanager webhook
type MachineChecker struct {
	clustercli      clusterapi.Interface
	arocli          aroclient.AroV1alpha1Interface
	log             *logrus.Entry
	developmentMode bool
	role            string
}

func NewMachineChecker(log *logrus.Entry, clustercli clusterapi.Interface, arocli aroclient.AroV1alpha1Interface, role string, developmentMode bool) *MachineChecker {
	return &MachineChecker{
		clustercli:      clustercli,
		arocli:          arocli,
		log:             log,
		role:            role,
		developmentMode: developmentMode,
	}
}

func (r *MachineChecker) workerReplicas() (int, error) {
	count := 0
	machinesets, err := r.clustercli.MachineV1beta1().MachineSets(machineSetsNamespace).List(metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	for _, machineset := range machinesets.Items {
		if machineset.Spec.Replicas != nil {
			count += int(*machineset.Spec.Replicas)
		}
	}
	return count, nil
}

func (r *MachineChecker) machineValid(ctx context.Context, machine *machinev1beta1.Machine) (bool, []string, error) {
	msgs := []string{}
	valid := true

	isMaster, err := utilmachine.IsMasterRole(machine)
	if err != nil {
		return true, nil, err
	}

	if machine.Spec.ProviderSpec.Value == nil {
		return true, nil, fmt.Errorf("provider spec is missing in the machine %q", machine.Name)
	}

	o, _, err := scheme.Codecs.UniversalDeserializer().Decode(machine.Spec.ProviderSpec.Value.Raw, nil, nil)
	if err != nil {
		return true, nil, err
	}

	machineProviderSpec, ok := o.(*azureproviderv1beta1.AzureMachineProviderSpec)
	if !ok {
		// This should never happen: codecs uses scheme that has only one registered type
		// and if something is wrong with the provider spec - decoding should fail
		return true, nil, fmt.Errorf("failed to read provider spec from the machine %q: %T", machine.Name, o)
	}

	if !utilmachine.VMSizeIsValid(api.VMSize(machineProviderSpec.VMSize), r.developmentMode, isMaster) {
		valid = false
		msgs = append(msgs, fmt.Sprintf("the machine %s VM size '%s' is invalid", machine.Name, machineProviderSpec.VMSize))
	}

	if !isMaster && !utilmachine.DiskSizeIsValid(machineProviderSpec.OSDisk.DiskSizeGB) {
		valid = false
		msgs = append(msgs, fmt.Sprintf("the machine %s disk size '%d' is invalid", machine.Name, machineProviderSpec.OSDisk.DiskSizeGB))
	}

	// to begin with, just check that the image publisher and offer are correct
	if machineProviderSpec.Image.Publisher != "azureopenshift" || machineProviderSpec.Image.Offer != "aro4" {
		valid = false
		msgs = append(msgs, fmt.Sprintf("the machine %s image '%v' is invalid", machine.Name, machineProviderSpec.Image))
	}

	if machineProviderSpec.ManagedIdentity != "" {
		valid = false
		msgs = append(msgs, fmt.Sprintf("the machine %s managedIdentity '%v' is invalid", machine.Name, machineProviderSpec.ManagedIdentity))
	}

	return valid, msgs, nil
}

func (r *MachineChecker) checkMachines(ctx context.Context) (bool, []string, error) {
	msgs := []string{}
	valid := true
	actualWorkers := 0
	actualMasters := 0

	machines, err := r.clustercli.MachineV1beta1().Machines(machineSetsNamespace).List(metav1.ListOptions{})
	if err != nil {
		return valid, msgs, err
	}
	for _, machine := range machines.Items {
		mValid, msgsMachine, err := r.machineValid(ctx, &machine)
		if err != nil {
			r.log.Errorf("machineValid err:%v", err)
			return valid, msgs, err
		}
		if !mValid {
			valid = false
			msgs = append(msgs, msgsMachine...)
		}
		isMaster, err := utilmachine.IsMasterRole(&machine)
		if err != nil {
			return valid, msgs, err
		}
		if isMaster {
			actualMasters++
		} else {
			actualWorkers++
		}
	}

	expectedMasters := 3
	expectedWorkers, err := r.workerReplicas()
	if err != nil {
		return valid, msgs, err
	}

	if actualMasters != expectedMasters {
		valid = false
		msgs = append(msgs, fmt.Sprintf("invalid number of master machines %d", actualMasters))
	}

	if actualWorkers != expectedWorkers {
		valid = false
		msgs = append(msgs, fmt.Sprintf("invalid number of worker machines %d, expected %d", actualWorkers, expectedWorkers))
	}
	return valid, msgs, nil
}

func (r *MachineChecker) Name() string {
	return "MachineChecker"
}

// Reconcile makes sure that the Machines are in a supportable state
func (r *MachineChecker) Check() error {
	ctx := context.Background()
	cond := &status.Condition{
		Type:    aro.MachineValid,
		Status:  corev1.ConditionTrue,
		Message: "all machines valid",
		Reason:  "CheckDone",
	}

	valid, msgs, err := r.checkMachines(ctx)
	if err != nil {
		r.log.Errorf("checkMachines err:%v", err)
		return err
	}
	if !valid {
		cond.Status = corev1.ConditionFalse
		cond.Reason = "CheckFailed"
		cond.Message = strings.Join(msgs, "\n")
	}

	return controllers.SetCondition(r.arocli, cond, r.role)
}
