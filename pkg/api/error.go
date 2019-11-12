package api

import (
	"fmt"
)

// CloudError represents a cloud error.
type CloudError struct {
	StatusCode     int `json:"-"`
	CloudErrorBody `json:"error,omitempty"`
}

func (err *CloudError) Error() string {
	return fmt.Sprintf("%d: %s: %s", err.StatusCode, err.Code, err.Message)
}

// CloudErrorBody represents the body of a cloud error.
type CloudErrorBody struct {
	Code    string           `json:"code,omitempty"`
	Message string           `json:"message,omitempty"`
	Target  string           `json:"target,omitempty"`
	Details []CloudErrorBody `json:"details,omitempty"`
}

// CloudErrorCodes
var (
	CloudErrorCodeInternalServerError      = "InternalServerError"
	CloudErrorCodeInvalidParameter         = "InvalidParameter"
	CloudErrorCodeInvalidRequestContent    = "InvalidRequestContent"
	CloudErrorCodeInvalidResource          = "InvalidResource"
	CloudErrorCodeInvalidResourceNamespace = "InvalidResourceNamespace"
	CloudErrorCodeInvalidResourceType      = "InvalidResourceType"
	CloudErrorCodeInvalidSubscriptionID    = "CloudErrorCodeInvalidSubscriptionID"
	CloudErrorCodeMismatchingResourceID    = "MismatchingResourceID"
	CloudErrorCodeMismatchingResourceName  = "MismatchingResourceName"
	CloudErrorCodeMismatchingResourceType  = "MismatchingResourceType"
	CloudErrorCodePropertyChangeNotAllowed = "PropertyChangeNotAllowed"
	CloudErrorCodeRequestNotAllowed        = "RequestNotAllowed"
	CloudErrorCodeResourceGroupNotFound    = "ResourceGroupNotFound"
	CloudErrorCodeResourceNotFound         = "ResourceNotFound"
	CloudErrorCodeUnsupportedMediaType     = "UnsupportedMediaType"
)

// NewCloudError returns a new CloudError
func NewCloudError(statusCode int, code, target, message string, a ...interface{}) *CloudError {
	return &CloudError{
		StatusCode: statusCode,
		CloudErrorBody: CloudErrorBody{
			Code:    code,
			Message: fmt.Sprintf(message, a...),
			Target:  target,
		},
	}
}
