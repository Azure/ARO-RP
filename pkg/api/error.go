package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// CloudError represents a cloud error.
type CloudError struct {
	// The status code.
	StatusCode int `json:"-"`

	// An error response from the service.
	*CloudErrorBody `json:"error,omitempty"`
}

func (err *CloudError) Error() string {
	var body string

	if err.CloudErrorBody != nil {
		body = ": " + err.CloudErrorBody.String()
	}

	return fmt.Sprintf("%d%s", err.StatusCode, body)
}

// CloudErrorBody represents the body of a cloud error.
type CloudErrorBody struct {
	// An identifier for the error. Codes are invariant and are intended to be consumed programmatically.
	Code string `json:"code,omitempty"`

	// A message describing the error, intended to be suitable for display in a user interface.
	Message string `json:"message,omitempty"`

	// The target of the particular error. For example, the name of the property in error.
	Target string `json:"target,omitempty"`

	//A list of additional details about the error.
	Details []CloudErrorBody `json:"details,omitempty"`
}

func (b *CloudErrorBody) String() string {
	var details string

	if len(b.Details) > 0 {
		details = " Details: "
		for i, innerErr := range b.Details {
			details += innerErr.String()
			if i < len(b.Details)-1 {
				details += ", "
			}
		}
	}

	return fmt.Sprintf("%s: %s: %s%s", b.Code, b.Target, b.Message, details)
}

// CloudErrorCodes
const (
	CloudErrorCodeInternalServerError                = "InternalServerError"
	CloudErrorCodeDeploymentFailed                   = "DeploymentFailed"
	CloudErrorCodeInvalidParameter                   = "InvalidParameter"
	CloudErrorCodeInvalidRequestContent              = "InvalidRequestContent"
	CloudErrorCodeInvalidResource                    = "InvalidResource"
	CloudErrorCodeDuplicateResourceGroup             = "DuplicateResourceGroup"
	CloudErrorCodeInvalidResourceNamespace           = "InvalidResourceNamespace"
	CloudErrorCodeInvalidResourceType                = "InvalidResourceType"
	CloudErrorCodeInvalidSubscriptionID              = "InvalidSubscriptionID"
	CloudErrorCodeMismatchingResourceID              = "MismatchingResourceID"
	CloudErrorCodeMismatchingResourceName            = "MismatchingResourceName"
	CloudErrorCodeMismatchingResourceType            = "MismatchingResourceType"
	CloudErrorCodePropertyChangeNotAllowed           = "PropertyChangeNotAllowed"
	CloudErrorCodeRequestNotAllowed                  = "RequestNotAllowed"
	CloudErrorCodeResourceGroupNotFound              = "ResourceGroupNotFound"
	CloudErrorCodeResourceNotFound                   = "ResourceNotFound"
	CloudErrorCodeUnsupportedMediaType               = "UnsupportedMediaType"
	CloudErrorCodeInvalidLinkedVNet                  = "InvalidLinkedVNet"
	CloudErrorCodeInvalidLinkedRouteTable            = "InvalidLinkedRouteTable"
	CloudErrorCodeInvalidLinkedDiskEncryptionSet     = "InvalidLinkedDiskEncryptionSet"
	CloudErrorCodeNotFound                           = "NotFound"
	CloudErrorCodeForbidden                          = "Forbidden"
	CloudErrorCodeInvalidSubscriptionState           = "InvalidSubscriptionState"
	CloudErrorCodeInvalidServicePrincipalCredentials = "InvalidServicePrincipalCredentials"
	CloudErrorCodeInvalidServicePrincipalClaims      = "InvalidServicePrincipalClaims"
	CloudErrorCodeInvalidResourceProviderPermissions = "InvalidResourceProviderPermissions"
	CloudErrorCodeInvalidServicePrincipalPermissions = "InvalidServicePrincipalPermissions"
	CloudErrorCodeInvalidLocation                    = "InvalidLocation"
	CloudErrorCodeInvalidOperationID                 = "InvalidOperationID"
	CloudErrorCodeDuplicateClientID                  = "DuplicateClientID"
	CloudErrorCodeDuplicateDomain                    = "DuplicateDomain"
	CloudErrorCodeResourceQuotaExceeded              = "ResourceQuotaExceeded"
	CloudErrorCodeQuotaExceeded                      = "QuotaExceeded"
	CloudErrorResourceProviderNotRegistered          = "ResourceProviderNotRegistered"
)

// NewCloudError returns a new CloudError
func NewCloudError(statusCode int, code, target, message string, a ...interface{}) *CloudError {
	return &CloudError{
		StatusCode: statusCode,
		CloudErrorBody: &CloudErrorBody{
			Code:    code,
			Message: fmt.Sprintf(message, a...),
			Target:  target,
		},
	}
}

// WriteError constructs and writes a CloudError to the given ResponseWriter
func WriteError(w http.ResponseWriter, statusCode int, code, target, message string, a ...interface{}) {
	WriteCloudError(w, NewCloudError(statusCode, code, target, message, a...))
}

// WriteCloudError writes a CloudError to the given ResponseWriter
func WriteCloudError(w http.ResponseWriter, err *CloudError) {
	w.WriteHeader(err.StatusCode)
	e := json.NewEncoder(w)
	e.SetIndent("", "    ")
	_ = e.Encode(err)
}
