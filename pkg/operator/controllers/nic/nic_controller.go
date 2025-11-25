package nic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/go-autorest/autorest/azure"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	ControllerName = "NIC"

	// Periodic reconciliation interval (safety net for orphaned NICs)
	periodicReconcileInterval = 1 * time.Hour
)

// Reconciler is the controller struct
type Reconciler struct {
	log    *logrus.Entry
	client client.Client
}

// reconcileManager is an instance of the manager instantiated per request
type reconcileManager struct {
	log            *logrus.Entry
	client         client.Client
	instance       *arov1alpha1.Cluster
	subscriptionID string
	resourceGroup  string
	infraID        string
	nicClient      armnetwork.InterfacesClient
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

// Reconcile reconciles failed NICs in the cluster
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Check if controller is enabled via operator flag
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.NICEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running NIC reconciliation controller")

	// Parse cluster resource details
	azEnv, err := azureclient.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		return reconcile.Result{}, err
	}

	resource, err := azure.ParseResourceID(instance.Spec.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create Azure credential
	credential, err := azidentity.NewDefaultAzureCredential(azEnv.DefaultAzureCredentialOptions())
	if err != nil {
		return reconcile.Result{}, err
	}

	options := azEnv.ArmClientOptions()

	// Create NIC client
	nicClient, err := armnetwork.NewInterfacesClient(resource.SubscriptionID, credential, options)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Extract resource group from cluster profile
	resourceGroupID := instance.Spec.ClusterResourceGroupID
	if resourceGroupID == "" {
		return reconcile.Result{}, fmt.Errorf("cluster resource group ID is empty")
	}
	resourceGroup := stringutils.LastTokenByte(resourceGroupID, '/')

	manager := &reconcileManager{
		log:            r.log,
		client:         r.client,
		instance:       instance,
		subscriptionID: resource.SubscriptionID,
		resourceGroup:  resourceGroup,
		infraID:        instance.Spec.InfraID,
		nicClient:      *nicClient,
	}

	// Determine reconciliation type based on trigger
	var reconErr error
	if request.Name == arov1alpha1.SingletonClusterName {
		// Periodic reconciliation (triggered by Cluster object)
		r.log.Info("running periodic NIC reconciliation (full scan)")
		reconErr = manager.reconcileAllNICs(ctx)
	} else {
		// Event-driven reconciliation (triggered by Machine/MachineSet)
		r.log.Infof("running event-driven NIC reconciliation for: %s", request.Name)
		reconErr = manager.reconcileNICForMachine(ctx, request.Name, request.Namespace)
	}

	if reconErr != nil {
		r.log.Errorf("NIC reconciliation failed: %v", reconErr)
		// Return with requeue after delay for retry
		return reconcile.Result{RequeueAfter: 5 * time.Minute}, reconErr
	}

	// Schedule next periodic reconciliation
	return reconcile.Result{RequeueAfter: periodicReconcileInterval}, nil
}

// reconcileNICForMachine reconciles NIC for a specific Machine (event-driven path)
func (rm *reconcileManager) reconcileNICForMachine(ctx context.Context, resourceName, namespace string) error {
	rm.log.Infof("reconciling NIC for resource: %s/%s", namespace, resourceName)

	// Get the Machine object to extract NIC name from provider spec
	machine := &machinev1beta1.Machine{}
	err := rm.client.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, machine)
	if err != nil {
		// If machine not found, it may have been deleted - try MachineSet
		if client.IgnoreNotFound(err) == nil {
			rm.log.Infof("machine %s not found, attempting to derive NIC from MachineSet", resourceName)
			return rm.reconcileNICsForMachineSet(ctx, resourceName, namespace)
		}
		return fmt.Errorf("failed to get machine: %w", err)
	}

	// Extract NIC name from Machine's provider spec
	nicName, err := extractNICNameFromMachine(machine)
	if err != nil {
		rm.log.Warnf("could not extract NIC name from machine %s: %v", resourceName, err)
		return nil // Skip if we can't determine the NIC
	}

	if nicName == "" {
		rm.log.Warnf("NIC name is empty for machine: %s", resourceName)
		return nil
	}

	return rm.reconcileNIC(ctx, nicName)
}

// reconcileNICsForMachineSet reconciles NICs for machines in a MachineSet
func (rm *reconcileManager) reconcileNICsForMachineSet(ctx context.Context, machineSetName, namespace string) error {
	// List all machines owned by this MachineSet
	machineList := &machinev1beta1.MachineList{}
	err := rm.client.List(ctx, machineList, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	var errors []string
	for _, machine := range machineList.Items {
		// Check if machine is owned by this MachineSet
		for _, ownerRef := range machine.OwnerReferences {
			if ownerRef.Kind == "MachineSet" && ownerRef.Name == machineSetName {
				nicName, err := extractNICNameFromMachine(&machine)
				if err != nil {
					rm.log.Warnf("could not extract NIC name from machine %s: %v", machine.Name, err)
					continue
				}

				if err := rm.reconcileNIC(ctx, nicName); err != nil {
					errors = append(errors, fmt.Sprintf("NIC %s: %v", nicName, err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to reconcile some NICs:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// reconcileAllNICs reconciles all NICs in the resource group (periodic safety net)
func (rm *reconcileManager) reconcileAllNICs(ctx context.Context) error {
	rm.log.Infof("scanning all NICs in resource group: %s", rm.resourceGroup)

	// List all NICs in the cluster resource group
	pager := rm.nicClient.NewListPager(rm.resourceGroup, nil)

	var errors []string
	nicCount := 0
	failedNICCount := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list NICs: %w", err)
		}

		for _, nic := range page.Value {
			if nic.Name == nil {
				continue
			}

			nicCount++

			// Filter: only reconcile NICs belonging to this cluster (by infraID prefix)
			if !strings.HasPrefix(*nic.Name, rm.infraID) {
				continue
			}

			// Check if NIC is in failed state
			if isNICInFailedState(nic) {
				failedNICCount++
				rm.log.Warnf("found NIC in failed state: %s, provisioning state: %s",
					*nic.Name, string(*nic.Properties.ProvisioningState))

				if err := rm.reconcileNIC(ctx, *nic.Name); err != nil {
					errors = append(errors, fmt.Sprintf("NIC %s: %v", *nic.Name, err))
				}
			}
		}
	}

	rm.log.Infof("periodic scan complete: checked %d NICs, found %d failed NICs",
		nicCount, failedNICCount)

	if len(errors) > 0 {
		return fmt.Errorf("failed to reconcile some NICs:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// reconcileNIC attempts to reconcile a specific failed NIC
func (rm *reconcileManager) reconcileNIC(ctx context.Context, nicName string) error {
	rm.log.Infof("attempting to reconcile NIC: %s", nicName)

	// Get current NIC state
	nic, err := rm.nicClient.Get(ctx, rm.resourceGroup, nicName, nil)
	if err != nil {
		// If NIC doesn't exist, nothing to reconcile
		if isNotFoundError(err) {
			rm.log.Infof("NIC %s does not exist (may have been deleted)", nicName)
			return nil
		}
		return fmt.Errorf("failed to get NIC %s: %w", nicName, err)
	}

	// Check if NIC is in failed state
	if !isNICInFailedState(&nic.Interface) {
		rm.log.Debugf("NIC %s is not in failed state (state: %s), skipping",
			nicName, string(*nic.Properties.ProvisioningState))
		return nil
	}

	rm.log.Warnf("NIC %s is in failed state: %s", nicName, string(*nic.Properties.ProvisioningState))

	// Reconciliation strategy: retry NIC creation/update
	// This triggers Azure to re-attempt provisioning
	poller, err := rm.nicClient.BeginCreateOrUpdate(ctx, rm.resourceGroup, nicName, nic.Interface, nil)
	if err != nil {
		return fmt.Errorf("failed to start NIC reconciliation for %s: %w", nicName, err)
	}

	// Wait for operation to complete
	result, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("NIC reconciliation failed for %s: %w", nicName, err)
	}

	rm.log.Infof("NIC %s reconciled successfully, new state: %s",
		nicName, string(*result.Properties.ProvisioningState))

	return nil
}

// SetupWithManager creates the controller with event-driven + periodic triggers
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// PERIODIC: Watch Cluster object (triggers periodic full reconciliation)
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(
			predicate.And(
				predicates.AROCluster,
				predicate.GenerationChangedPredicate{},
			),
		)).
		// EVENT-DRIVEN: Watch Master Machine changes
		Watches(
			&source.Kind{Type: &machinev1beta1.Machine{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicates.MachineRoleMaster),
		).
		// EVENT-DRIVEN: Watch Worker MachineSet changes
		Watches(
			&source.Kind{Type: &machinev1beta1.MachineSet{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicates.MachineRoleWorker),
		).
		Named(ControllerName).
		Complete(r)
}

// Helper functions

// extractNICNameFromMachine extracts NIC name from Machine's Azure provider spec
func extractNICNameFromMachine(machine *machinev1beta1.Machine) (string, error) {
	if machine.Spec.ProviderSpec.Value == nil {
		return "", fmt.Errorf("machine provider spec is nil")
	}

	// Unmarshal Azure provider spec
	var providerSpec machinev1beta1.AzureMachineProviderSpec
	if err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, &providerSpec); err != nil {
		return "", fmt.Errorf("failed to unmarshal provider spec: %w", err)
	}

	// Azure Machine API creates NICs with pattern: <machine-name>-nic
	// The machine name is used as the VM name and Azure appends -nic for the NIC
	if machine.Name != "" {
		return machine.Name + "-nic", nil
	}

	return "", fmt.Errorf("could not determine NIC name from machine")
}

// isNICInFailedState checks if NIC provisioning state indicates failure
func isNICInFailedState(nic *armnetwork.Interface) bool {
	if nic.Properties == nil || nic.Properties.ProvisioningState == nil {
		return false
	}

	state := string(*nic.Properties.ProvisioningState)

	// Failed states as per Azure provisioning state enum
	failedStates := []string{
		"Failed",
		"Canceled", // Sometimes indicates partial failure
	}

	for _, failedState := range failedStates {
		if strings.EqualFold(state, failedState) {
			return true
		}
	}

	return false
}

// isNotFoundError checks if error is a 404 not found
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check Azure SDK specific error responses
	errStr := err.Error()
	return strings.Contains(errStr, "ResourceNotFound") ||
		strings.Contains(errStr, "NotFound") ||
		strings.Contains(errStr, "404")
}
