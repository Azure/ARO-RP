package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

const OperationPreemptedCode = "OperationPreempted"

func isDeploymentNotFoundError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "DeploymentNotFound" {
			return true
		}
	}
	return false
}

func isOperationPreemptedError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == OperationPreemptedCode {
			return true
		}
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok &&
			serviceErr.Code == OperationPreemptedCode {
			return true
		}
	}
	// Also check error message for OperationPreempted
	return err != nil && strings.Contains(err.Error(), OperationPreemptedCode)
}
