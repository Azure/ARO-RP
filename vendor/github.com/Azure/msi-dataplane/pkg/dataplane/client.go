package dataplane

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/msi-dataplane/pkg/dataplane/internal"
)

// Client wraps the generated code to smooth over the rough edges from generation, namely:
// - the generated clients incorrectly expose the API version as a parameter, even though there's only one option
// - the generated clients incorrectly expose all sorts of internal logic like the request body, etc, when we just want a clean client
// Ideally we wouldn't need this wrapper, but it's much easier to implement this here than update the generator.

// Client exposes the API for the MSI data plane.
type Client interface {
	// DeleteSystemAssignedIdentity deletes the system-assigned identity for a proxy resource.
	DeleteSystemAssignedIdentity(ctx context.Context) error

	// GetSystemAssignedIdentityCredentials retrieves the credentials for the system-assigned identity associated with the proxy resource.
	GetSystemAssignedIdentityCredentials(ctx context.Context) (*ManagedIdentityCredentials, error)

	// GetUserAssignedIdentitiesCredentials retrieves the credentials for any user-assigned identities associated with the proxy resource.
	GetUserAssignedIdentitiesCredentials(ctx context.Context, request UserAssignedIdentitiesRequest) (*ManagedIdentityCredentials, error)

	// MoveIdentity moves the identity from one resource group into another.
	MoveIdentity(ctx context.Context, request MoveIdentityRequest) (*MoveIdentityResponse, error)
}

var apiVersion string

func init() {
	date, err := time.Parse(time.RFC3339, string(internal.DeleteidentityParamsApiVersionN20240101T000000Z))
	if err != nil {
		panic(fmt.Errorf("failed to parse generated API version as date: %s", err.Error()))
	}
	apiVersion = date.Format("2006-01-02")
}

type clientAdapter struct {
	delegate *internal.ClientWithResponses
}

var _ Client = (*clientAdapter)(nil)

func (c *clientAdapter) DeleteSystemAssignedIdentity(ctx context.Context) error {
	resp, err := c.delegate.DeleteidentityWithResponse(ctx, &internal.DeleteidentityParams{ApiVersion: internal.DeleteidentityParamsApiVersion(apiVersion)})
	if err != nil {
		return err
	}
	for _, respErr := range []*internal.ErrorResponse{resp.JSON400, resp.JSON401, resp.JSON403, resp.JSON404, resp.JSON405, resp.JSON429, resp.JSON500, resp.JSON503} {
		if respErr != nil {
			return &ResponseError{WrappedError: *respErr}
		}
	}
	return nil
}

func (c *clientAdapter) GetSystemAssignedIdentityCredentials(ctx context.Context) (*ManagedIdentityCredentials, error) {
	resp, err := c.delegate.GetcredWithResponse(ctx, &internal.GetcredParams{ApiVersion: internal.GetcredParamsApiVersion(apiVersion)})
	if err != nil {
		return nil, err
	}
	for _, respErr := range []*internal.ErrorResponse{resp.JSON400, resp.JSON401, resp.JSON403, resp.JSON404, resp.JSON429, resp.JSON500, resp.JSON503} {
		if respErr != nil {
			return nil, &ResponseError{WrappedError: *respErr}
		}
	}
	return resp.JSON200, nil
}

func (c *clientAdapter) GetUserAssignedIdentitiesCredentials(ctx context.Context, request UserAssignedIdentitiesRequest) (*ManagedIdentityCredentials, error) {
	resp, err := c.delegate.GetcredsWithResponse(ctx, &internal.GetcredsParams{ApiVersion: internal.GetcredsParamsApiVersion(apiVersion)}, request)
	if err != nil {
		return nil, err
	}
	for _, respErr := range []*internal.ErrorResponse{resp.JSON400, resp.JSON401, resp.JSON403, resp.JSON404, resp.JSON405, resp.JSON429, resp.JSON500, resp.JSON503} {
		if respErr != nil {
			return nil, &ResponseError{WrappedError: *respErr}
		}
	}
	return resp.JSON200, nil
}

func (c *clientAdapter) MoveIdentity(ctx context.Context, request MoveIdentityRequest) (*MoveIdentityResponse, error) {
	resp, err := c.delegate.MoveidentityWithResponse(ctx, &internal.MoveidentityParams{ApiVersion: internal.MoveidentityParamsApiVersion(apiVersion)}, request)
	if err != nil {
		return nil, err
	}
	for _, respErr := range []*internal.ErrorResponse{resp.JSON400, resp.JSON401, resp.JSON403, resp.JSON404, resp.JSON405, resp.JSON429, resp.JSON500, resp.JSON503} {
		if respErr != nil {
			return nil, &ResponseError{WrappedError: *respErr}
		}
	}
	return resp.JSON200, nil
}
