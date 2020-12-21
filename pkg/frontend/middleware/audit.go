package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Audit generates an audit log based on the request, caller, resource and
// correlation data found in a HTTP request. It depends on the 'Log' middleware
// to populate the request context with data that it needs.
func Audit(env env.Core, entry *logrus.Entry) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// skip requests to the 'operationsstatus' endpoints
			if strings.Contains(r.URL.Path, "operationsstatus") {
				h.ServeHTTP(w, r)
				return
			}

			callerIdentity, callerType, correlationID, requestID := callerRequestData(r)
			targetResourceName, targetResourceType := targetResourceData(entry, r)

			auditEntry := audit.NewEntry(entry.Logger)
			auditEntry = auditEntry.WithFields(logrus.Fields{
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
				auditEntry.WithFields(logrus.Fields{
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
	callerIdentity string,
	callerType string,
	correlationID string,
	requestID string) {

	callerIdentity = r.UserAgent()
	callerType = audit.CallerIdentityTypeApplicationID

	if v := r.Context().Value(ContextKeyCorrelationData); v != nil {
		if correlationData, ok := v.(*api.CorrelationData); ok {
			if correlationData.ClientPrincipalName != "" {
				callerIdentity = correlationData.ClientPrincipalName
				callerType = audit.CallerIdentityTypeObjectID
			}

			correlationID = correlationData.CorrelationID
			requestID = correlationData.RequestID
		}
	}

	return
}

func targetResourceData(entry *logrus.Entry, r *http.Request) (string, string) {
	entry = log.EnrichWithPath(entry, r.URL.Path)

	var resourceName, resourceKind string
	if v, ok := entry.Data["resource_id"].(string); ok {
		resourceName = v
	}

	if v, ok := entry.Data["resource_kind"].(string); ok {
		resourceKind = v
	}

	if resourceKind == "" && strings.Contains(r.URL.Path, "/admin") {
		resourceKind = audit.ResourceTypeAdminAction
	}

	return resourceName, resourceKind
}

func resultData(w http.ResponseWriter) (string, string) {
	var (
		resultType        = audit.ResultTypeSuccess
		resultDescription string
	)

	statusCode := http.StatusOK
	if v, ok := w.(*logResponseWriter); ok {
		statusCode = v.statusCode
	}
	if statusCode >= http.StatusBadRequest {
		resultType = audit.ResultTypeFail
	}
	resultDescription = fmt.Sprintf("Status code: %d", statusCode)

	return resultType, resultDescription
}
