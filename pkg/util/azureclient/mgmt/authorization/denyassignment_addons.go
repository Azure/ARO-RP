package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	policy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	runtime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	"github.com/Azure/ARO-RP/pkg/api"
)

// DenyAssignmentClientAddons contains addons for DenyAssignmentClient
type DenyAssignmentClientAddons interface {
	ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []mgmtauthorization.DenyAssignment, err error)
	DeleteDenyAssignment(ctx context.Context, fpTokenCredential *azidentity.ClientCertificateCredential, subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error
}

func (c *denyAssignmentClient) ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []mgmtauthorization.DenyAssignment, err error) {
	page, err := c.DenyAssignmentsClient.ListForResourceGroup(ctx, resourceGroupName, filter)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)
		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *denyAssignmentClient) DeleteDenyAssignment(ctx context.Context, fpTokenCredential *azidentity.ClientCertificateCredential, subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error {
	deleteClient, err := NewDenyAssignmentsARMClient(subscriptionDoc.ID, fpTokenCredential, nil)
	if err != nil {
		return err
	}

	managedResourceGroupName := strings.Split(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, "/")[3]
	denyAssignmentList, err := c.ListForResourceGroup(ctx, managedResourceGroupName, "")
	if err != nil {
		return err
	}

	// There should never be more than one deny assignment, but in case there is, don't proceed
	if len(denyAssignmentList) > 1 {
		return nil
	}
	denyAssignmentID := denyAssignmentList[0].ID

	// Delete the deny assignment by ID
	delete, err := c.deleteDenyAssignmentRequest(ctx, *deleteClient, *denyAssignmentID)
	if err != nil {
		return err
	}
	d, err := deleteClient.internal.Pipeline().Do(delete)
	if err != nil {
		return err
	}
	defer d.Body.Close()

	return nil
}

// getCreateRequest creates the deny assignment delete request.
func (c *denyAssignmentClient) deleteDenyAssignmentRequest(ctx context.Context, client DenyAssignmentsARMClient, denyAssignmentID string) (*policy.Request, error) {
	urlPath := denyAssignmentID

	req, err := runtime.NewRequest(ctx, http.MethodGet, runtime.JoinPaths(client.internal.Endpoint(), urlPath))
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", "2022-04-01")
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header["Accept"] = []string{"application/json"}
	return req, nil
}
