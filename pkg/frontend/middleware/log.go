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

	"github.com/microsoft/go-otel-audit/audit/msgs"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	otelaudit "github.com/Azure/ARO-RP/pkg/util/log/audit/otel_audit"
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

type LogMiddleware struct {
	EnvironmentName string
	Hostname        string
	Location        string
	AuditLog        *logrus.Entry
	OtelAudit       *otelaudit.Audit
	BaseLog         *logrus.Entry
}

func (l LogMiddleware) Log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		r.Body = &logReadCloser{ReadCloser: r.Body}
		w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		correlationData := api.CreateCorrelationDataFromReq(r)
		correlationData.RequestTime = t

		if r.URL.Query().Get(api.APIVersionKey) == admin.APIVersion || isAdminOp(r) {
			correlationData.ClientPrincipalName = r.Header.Get("X-Ms-Client-Principal-Name")
		}

		w.Header().Set("X-Ms-Request-Id", correlationData.RequestID)

		if strings.EqualFold(r.Header.Get("X-Ms-Return-Client-Request-Id"), "true") {
			w.Header().Set("X-Ms-Client-Request-Id", correlationData.ClientRequestID)
		}

		log := l.BaseLog
		log = utillog.EnrichWithPath(log, r.URL.Path)
		log = utillog.EnrichWithCorrelationData(log, correlationData)

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyLog, log)
		ctx = api.CtxWithCorrelationData(ctx, correlationData)

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
			auditCallerIdentity = r.UserAgent()
			auditCallerType     = audit.CallerIdentityTypeApplicationID
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

		auditEntry := l.AuditLog.WithFields(logrus.Fields{
			audit.MetadataCreatedTime:     logTime,
			audit.MetadataLogKind:         audit.IFXAuditLogKind,
			audit.MetadataSource:          audit.SourceRP,
			audit.MetadataAdminOperation:  adminOp,
			audit.EnvKeyAppID:             audit.SourceRP,
			audit.EnvKeyCloudRole:         audit.CloudRoleRP,
			audit.EnvKeyCorrelationID:     correlationData.CorrelationID,
			audit.EnvKeyEnvironment:       l.EnvironmentName,
			audit.EnvKeyHostname:          l.Hostname,
			audit.EnvKeyLocation:          l.Location,
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
					TargetResourceName: r.URL.Path,
					TargetResourceType: auditTargetResourceType(r),
				},
			},
		})

		auditMsg := msgs.Msg{Type: msgs.ControlPlane}
		auditRec := otelaudit.GetAuditRecord()

		callerIpAddress, err := msgs.ParseAddr(r.RemoteAddr)
		if err != nil {
			log.Printf("Error parsing remote address: %s, error: %v", r.RemoteAddr, err)
		}

		auditRec{
			CallerIpAddress: callerIpAddress,
			CallerIdentities: map[msgs.CallerIdentityType][]msgs.CallerIdentityEntry{
				msgs.ApplicationID: []msgs.CallerIdentityEntry{
					{
						Identity:    audit.CallerIdentityTypeApplicationID,
						Description: "Client application ID",
					},
				},
				// Need to revisit this
				// msgs.ObjectID: []msgs.CallerIdentityEntry{
				// 	{
				// 		Identity:    audit.CallerIdentityTypeObjectID,
				// 		Description: "Client Object ID",
				// 	},
				// },
				// msgs.UPN: []msgs.CallerIdentityEntry{
				// 	{
				// 		Identity:    correlationData.ClientPrincipalName,
				// 		Description: "Client principal name",
				// 	},
				// },
			},
			OperationCategories: []msgs.OperationCategory{msgs.ResourceManagement},
			CustomData:          otelaudit.GetCustomData(),
			TargetResources: map[string][]msgs.TargetResourceEntry{
				auditTargetResourceType(r): []msgs.TargetResourceEntry{
					{
						Name:   r.URL.Path,
						Region: l.Location,
					},
				},
			},
			CallerAccessLevels:   "Caller admin AccessLevels",
			OperationAccessLevel: "Portal Admin Operation AccessLevel",
			OperationName:        operationName,
			// OperationResultDescription: fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			CallerAgent:                  r.UserAgent(),
			OperationCategoryDescription: "Client Resource Management via frontend",
			OperationType:                otelaudit.GetOperationType(r.Method),
			// OperationResult:
		}

		defer func() {
			statusCode := w.(*logResponseWriter).statusCode
			log.WithFields(logrus.Fields{
				"body_read_bytes":      r.Body.(*logReadCloser).bytes,
				"body_written_bytes":   w.(*logResponseWriter).bytes,
				"duration":             time.Since(t).Seconds(),
				"response_status_code": statusCode,
			}).Print("sent response")

			resultType := audit.ResultTypeSuccess
			auditRec.OperationResult = msgs.Success

			if statusCode >= http.StatusBadRequest {
				resultType = audit.ResultTypeFail
				auditRec.OperationResult = msgs.Failure
				auditRec.OperationResultDescription = fmt.Sprintf("Status code: %d", statusCode)
			}

			if r.URL.Path == "/healthz/ready" {
				return
			}

			auditEntry.WithFields(logrus.Fields{
				audit.PayloadKeyResult: audit.Result{
					ResultType:        resultType,
					ResultDescription: fmt.Sprintf("Status code: %d", statusCode),
				},
			}).Info(audit.DefaultLogMessage)

			auditMsg.Record = auditRec

			if err := l.OtelAudit.Client.SendAuditMessage(r.Context(), &auditMsg); err != nil {
				log.Printf("Frontend - Error sending audit message: %v", err)
			}
		}()

		h.ServeHTTP(w, r)
	})
}

func auditTargetResourceType(r *http.Request) string {
	if matches := utillog.RXProviderResourceKind.FindStringSubmatch(r.URL.Path); matches != nil {
		return matches[len(matches)-1]
	}

	if matches := utillog.RXAdminProvider.FindStringSubmatch(r.URL.Path); matches != nil {
		return matches[len(matches)-1]
	}

	if matches := utillog.RXTolerantResourceID.FindStringSubmatch(r.URL.Path); matches != nil {
		return matches[len(matches)-2]
	}

	return ""
}

func isAdminOp(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/admin")
}
