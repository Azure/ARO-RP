package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

type logResponseWriter struct {
	http.ResponseWriter

	statusCode int
	bytes      int
}

func (w *logResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *logResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type logReadCloser struct {
	io.ReadCloser

	bytes int
}

func (rc *logReadCloser) Read(b []byte) (int, error) {
	n, err := rc.ReadCloser.Read(b)
	rc.bytes += n
	return n, err
}

func Log(env env.Core, auditLog, baseLog *logrus.Entry) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now()

			vars := mux.Vars(r)

			r.Body = &logReadCloser{ReadCloser: r.Body}
			w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			correlationData := &api.CorrelationData{
				ClientRequestID: r.Header.Get("X-Ms-Client-Request-Id"),
				CorrelationID:   r.Header.Get("X-Ms-Correlation-Request-Id"),
				RequestID:       uuid.NewV4().String(),
				RequestTime:     t,
			}

			if vars["api-version"] == admin.APIVersion || isAdminOp(r) {
				correlationData.ClientPrincipalName = r.Header.Get("X-Ms-Client-Principal-Name")
			}

			w.Header().Set("X-Ms-Request-Id", correlationData.RequestID)

			if strings.EqualFold(r.Header.Get("X-Ms-Return-Client-Request-Id"), "true") {
				w.Header().Set("X-Ms-Client-Request-Id", correlationData.ClientRequestID)
			}

			log := baseLog
			log = utillog.EnrichWithPath(log, r.URL.Path)
			log = utillog.EnrichWithCorrelationData(log, correlationData)

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyLog, log)
			ctx = context.WithValue(ctx, ContextKeyCorrelationData, correlationData)

			r = r.WithContext(ctx)

			log = log.WithFields(logrus.Fields{
				"request_method":      r.Method,
				"request_path":        r.URL.Path,
				"request_proto":       r.Proto,
				"request_remote_addr": r.RemoteAddr,
				"request_user_agent":  r.UserAgent(),
			})
			log.Print("read request")

			var (
				auditCallerIdentity                              = r.UserAgent()
				auditCallerType                                  = audit.CallerIdentityTypeApplicationID
				auditTargetResourceName, auditTargetResourceType = auditTargetResourceData(r)
			)

			if correlationData.ClientPrincipalName != "" {
				auditCallerIdentity = correlationData.ClientPrincipalName
				auditCallerType = audit.CallerIdentityTypeObjectID
			}

			var (
				adminOp       = isAdminOp(r)
				logTime       = time.Now().UTC().Format(time.RFC3339)
				operationName = fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			)

			auditEntry := auditLog.WithFields(logrus.Fields{
				audit.MetadataCreatedTime:     logTime,
				audit.MetadataLogKind:         audit.IFXAuditLogKind,
				audit.MetadataSource:          audit.SourceRP,
				audit.MetadataAdminOperation:  adminOp,
				audit.EnvKeyAppID:             audit.SourceRP,
				audit.EnvKeyCloudRole:         audit.CloudRoleRP,
				audit.EnvKeyCorrelationID:     correlationData.CorrelationID,
				audit.EnvKeyEnvironment:       env.Environment().Name,
				audit.EnvKeyHostname:          env.Hostname(),
				audit.EnvKeyLocation:          env.Location(),
				audit.PayloadKeyCategory:      audit.CategoryResourceManagement,
				audit.PayloadKeyOperationName: operationName,
				audit.PayloadKeyRequestID:     correlationData.RequestID,
				audit.PayloadKeyCallerIdentities: []audit.CallerIdentity{
					{
						CallerIdentityType:  auditCallerType,
						CallerIdentityValue: auditCallerIdentity,
						CallerIPAddress:     r.RemoteAddr,
					},
				},
				audit.PayloadKeyTargetResources: []audit.TargetResource{
					{
						TargetResourceName: auditTargetResourceName,
						TargetResourceType: auditTargetResourceType,
					},
				},
			})

			defer func() {
				statusCode := w.(*logResponseWriter).statusCode
				log.WithFields(logrus.Fields{
					"body_read_bytes":      r.Body.(*logReadCloser).bytes,
					"body_written_bytes":   w.(*logResponseWriter).bytes,
					"duration":             time.Since(t).Seconds(),
					"response_status_code": statusCode,
				}).Print("sent response")

				resultType := audit.ResultTypeSuccess
				if statusCode >= http.StatusBadRequest {
					resultType = audit.ResultTypeFail
				}

				auditEntry.WithFields(logrus.Fields{
					audit.PayloadKeyResult: audit.Result{
						ResultType:        resultType,
						ResultDescription: fmt.Sprintf("Status code: %d", statusCode),
					},
				}).Info(audit.DefaultLogMessage)
			}()

			h.ServeHTTP(w, r)
		})
	}
}

func auditTargetResourceData(r *http.Request) (string, string) {
	if matches := utillog.RXProviderResourceKind.FindStringSubmatch(r.URL.Path); matches != nil {
		if resourceKind := matches[len(matches)-1]; resourceKind != "" {
			return resourceKind, ""
		}
	}

	if matches := utillog.RXAdminProvider.FindStringSubmatch(r.URL.Path); matches != nil {
		if resourceKind := matches[len(matches)-1]; resourceKind != "" {
			return resourceKind, ""
		}
	}

	if matches := utillog.RXTolerantResourceID.FindStringSubmatch(r.URL.Path); matches != nil {
		resourceKind, resourceName := matches[len(matches)-2], matches[len(matches)-1]
		return resourceKind, resourceName
	}

	return "", ""
}

func isAdminOp(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/admin")
}
