package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	policy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	runtime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	"github.com/Azure/ARO-RP/pkg/api"
)

// DenyAssignmentClientAddons contains addons for DenyAssignmentClient
type DenyAssignmentClientAddons interface {
	ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []authorization.DenyAssignment, err error)
	DeleteDenyAssignment(ctx context.Context, fpTokenCredential *azidentity.ClientCertificateCredential, subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error
}

func (c *denyAssignmentClient) ListForResourceGroup(ctx context.Context, resourceGroupName string, filter string) (result []authorization.DenyAssignment, err error) {
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

// getCreateRequest creates the deny assignment get request.
func (c *denyAssignmentClient) getDenyAssignmentRequest(ctx context.Context, client DenyAssignmentsARMClient, subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) (*policy.Request, error) {
	urlPath := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/microsoft.authorization/denyassignments", subscriptionDoc.ID, doc.ClusterResourceGroupIDKey)

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

func (c *denyAssignmentClient) DeleteDenyAssignment(ctx context.Context, fpTokenCredential *azidentity.ClientCertificateCredential, subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error {
	client, err := NewDenyAssignmentsARMClient(subscriptionDoc.ID, fpTokenCredential, nil)
	if err != nil {
		return err
	}

	// Get the deny assignment
	var denyAssignment authorization.DenyAssignment
	get, err := c.getDenyAssignmentRequest(ctx, *client, subscriptionDoc, doc)
	if err != nil {
		return err
	}
	httpResp, err := client.internal.Pipeline().Do(get)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	// Marshall response into type authorization.DenyAssignment
	err = json.NewDecoder(httpResp.Body).Decode(&denyAssignment)
	if err != nil {
		return err
	}

	// Delete the deny assignment by ID
	delete, err := c.deleteDenyAssignmentRequest(ctx, *client, *denyAssignment.ID)
	if err != nil {
		return err
	}
	d, err := client.internal.Pipeline().Do(delete)
	if err != nil {
		return err
	}
	defer d.Body.Close()

	return nil
}
