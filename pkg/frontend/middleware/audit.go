package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

// Audit generates an audit log based on the request, caller, resource and
// correlation data found in a HTTP request. It depends on the 'Log' middleware
// to populate the request context with data that it needs.
func Audit(env env.Core, entry *logrus.Entry) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callerIdentity, callerType, correlationID, requestID := callerRequestData(r)
			targetResourceName, targetResourceType := targetResourceData(entry, r)

			entry = entry.WithFields(logrus.Fields{
				audit.EnvKeyAppID:             audit.SourceRP,
				audit.EnvKeyCloudRole:         audit.CloudRoleRP,
				audit.EnvKeyCorrelationID:     correlationID,
				audit.EnvKeyEnvironment:       env.Environment().Name,
				audit.EnvKeyHostname:          env.Hostname(),
				audit.EnvKeyLocation:          env.Location(),
				audit.PayloadKeyCategory:      audit.CategoryResourceManagement,
				audit.PayloadKeyOperationName: fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				audit.PayloadKeyRequestID:     requestID,
				audit.PayloadKeyCallerIdentities: []audit.CallerIdentity{
					{
						CallerIdentityType:  callerType,
						CallerIdentityValue: callerIdentity,
						CallerIPAddress:     r.RemoteAddr,
					},
				},
				audit.PayloadKeyTargetResources: []audit.TargetResource{
					{
						TargetResourceName: targetResourceName,
						TargetResourceType: targetResourceType,
					},
				},
			})

			defer func() {
				resultType, resultDescription := resultData(w)
				entry.WithFields(logrus.Fields{
					audit.PayloadKeyResult: audit.Result{
						ResultType:        resultType,
						ResultDescription: resultDescription,
					},
				}).Info("audit event")
			}()

			h.ServeHTTP(w, r)
		})
	}
}

func callerRequestData(r *http.Request) (
	string, string, string, string) {

	var (
		callerIdentity = r.UserAgent()
		callerType     = audit.CallerIdentityTypeApplicationID
		correlationID  = ""
		requestID      = ""
	)

	// if log middleware isn't run, this type assertion will panic as intended.
	// the frontend will recover from the panic and exit
	correlationData := r.Context().Value(ContextKeyCorrelationData).(*api.CorrelationData)
	if correlationData != nil {
		if correlationData.ClientPrincipalName != "" {
			callerIdentity = correlationData.ClientPrincipalName
			callerType = audit.CallerIdentityTypeObjectID
		}

		correlationID = correlationData.CorrelationID
		requestID = correlationData.RequestID
	}

	return callerIdentity, callerType, correlationID, requestID
}

func targetResourceData(entry *logrus.Entry, r *http.Request) (string, string) {
	matches := utillog.RXTolerantResourceID.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		return audit.UnknownValue, audit.UnknownValue
	}

	var resourceName, resourceKind string
	if matches[3] != "" {
		resourceKind = matches[3]
	}

	if matches[5] != "" {
		resourceName = fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s", matches[1], matches[2], matches[3], matches[4], matches[5])
	}

	return resourceName, resourceKind
}

func resultData(w http.ResponseWriter) (string, string) {
	var (
		resultType        = audit.ResultTypeSuccess
		resultDescription string
	)

	statusCode := http.StatusOK

	// if log middleware isn't run, this type assertion will panic as intended.
	// the frontend will recover from the panic and exit
	responseWriter := w.(*logResponseWriter)
	if responseWriter != nil {
		statusCode = responseWriter.statusCode
	}

	if statusCode >= http.StatusBadRequest {
		resultType = audit.ResultTypeFail
	}
	resultDescription = fmt.Sprintf("Status code: %d", statusCode)

	return resultType, resultDescription
}
