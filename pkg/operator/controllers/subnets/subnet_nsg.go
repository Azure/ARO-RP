package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/subnet"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

const (
	AnnotationTimestamp = "aro.openshift.io/lastSubnetReconcileTimestamp"
)

// ensureSubnetNSG verifies the subnet has the correct Network Security Group assigned.
// If the NSG is missing or incorrect, it updates the subnet with the correct NSG
// and records the reconciliation timestamp on the Cluster resource.
func (r *reconcileManager) ensureSubnetNSG(ctx context.Context, s subnet.Subnet) error {
	architectureVersion := api.ArchitectureVersion(r.instance.Spec.ArchitectureVersion)

	subnetID, err := arm.ParseResourceID(s.ResourceID)
	if err != nil {
		return err
	}

	subnetObject, err := r.subnets.Get(ctx, subnetID.ResourceGroupName, subnetID.Parent.Name, subnetID.Name, nil)
	if err != nil {
		if azureerrors.IsNotFoundError(err) {
			r.log.Infof("Subnet %s not found, skipping", s.ResourceID)
			return nil
		}
		return err
	}
	if subnetObject.Properties == nil {
		return fmt.Errorf("received nil, expected a value in subnetProperties when trying to Get subnet %s", s.ResourceID)
	}

	correctNSGResourceID, err := subnet.NetworkSecurityGroupIDExpanded(architectureVersion, r.instance.Spec.ClusterResourceGroupID, r.instance.Spec.InfraID, !s.IsMaster)
	if err != nil {
		return err
	}

	// if the NSG is assigned && it's the correct NSG - do nothing
	if subnetObject.Properties.NetworkSecurityGroup != nil &&
		subnetObject.Properties.NetworkSecurityGroup.ID != nil &&
		strings.EqualFold(*subnetObject.Properties.NetworkSecurityGroup.ID, correctNSGResourceID) {
		return nil
	}

	// else the NSG assignment needs to be corrected
	oldNSG := "nil"
	if subnetObject.Properties.NetworkSecurityGroup != nil {
		oldNSG = *subnetObject.Properties.NetworkSecurityGroup.ID
	}
	r.log.Infof("Fixing NSG from %s to %s", oldNSG, correctNSGResourceID)
	subnetObject.Properties.NetworkSecurityGroup = &armnetwork.SecurityGroup{ID: &correctNSGResourceID}
	err = r.subnets.CreateOrUpdateAndWait(ctx, subnetID.ResourceGroupName, subnetID.Parent.Name, subnetID.Name, subnetObject.Subnet, nil)
	if err != nil {
		return err
	}

	return r.updateReconcileSubnetAnnotation(ctx)
}

// updateReconcileSubnetAnnotation updates the Cluster resource with a timestamp annotation
// indicating when the last subnet reconciliation occurred. It uses retry-on-conflict to
// handle concurrent modifications to the Cluster resource.
func (r *reconcileManager) updateReconcileSubnetAnnotation(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cluster := &arov1alpha1.Cluster{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster); err != nil {
			return err
		}

		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations[AnnotationTimestamp] = time.Now().Format(time.RFC1123)

		patchPayload := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: cluster.Annotations,
			},
		}
		payloadBytes, err := json.Marshal(patchPayload)
		if err != nil {
			return err
		}
		return r.client.Patch(ctx, cluster, client.RawPatch(types.MergePatchType, payloadBytes))
	})
}
