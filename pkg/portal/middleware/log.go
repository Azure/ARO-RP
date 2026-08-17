package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/microsoft/go-otel-audit/audit/msgs"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

type logResponseWriter struct {
	http.ResponseWriter

	statusCode int
	bytes      int
}

func (w *logResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker := w.ResponseWriter.(http.Hijacker)
	return hijacker.Hijack()
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

func Log(env env.Core, auditLog, baseLog *logrus.Entry, outelAuditClient audit.Client) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now()

			r.Body = &logReadCloser{ReadCloser: r.Body}
			w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Prevent browsers from MIME-sniffing responses into HTML,
			// mitigating reflected XSS if any user-controlled content is echoed
			// back.
			w.Header().Set("X-Content-Type-Options", "nosniff")

			log := baseLog
			log = utillog.EnrichWithPath(log, r.URL.Path)

			username, _ := r.Context().Value(ContextKeyUsername).(string)
			username = utillog.Sanitize(username)

			log = log.WithFields(logrus.Fields{
				"request_method":      utillog.Sanitize(r.Method),
				"request_path":        utillog.Sanitize(r.URL.Path),
				"request_proto":       utillog.Sanitize(r.Proto),
				"request_remote_addr": utillog.Sanitize(r.RemoteAddr),
				"request_user_agent":  utillog.Sanitize(r.UserAgent()),
				"username":            username,
			})
			log.Print("read request")

			auditEntry := auditLog.WithFields(logrus.Fields{
				audit.MetadataAdminOperation:  true,
				audit.MetadataCreatedTime:     time.Now().UTC().Format(time.RFC3339),
				audit.MetadataLogKind:         audit.IFXAuditLogKind,
				audit.MetadataSource:          audit.SourceAdminPortal,
				audit.EnvKeyAppID:             audit.SourceAdminPortal,
				audit.EnvKeyCloudRole:         audit.CloudRoleRP,
				audit.EnvKeyEnvironment:       env.Environment().Name,
				audit.EnvKeyHostname:          env.Hostname(),
				audit.EnvKeyLocation:          env.Location(),
				audit.PayloadKeyCategory:      audit.CategoryResourceManagement,
				audit.PayloadKeyOperationName: fmt.Sprintf("%s %s", utillog.Sanitize(r.Method), utillog.Sanitize(r.URL.Path)),
				audit.PayloadKeyCallerIdentities: []audit.CallerIdentity{
					{
						CallerIdentityType:  audit.CallerIdentityTypeUsername,
						CallerIdentityValue: username,
						CallerIPAddress:     utillog.Sanitize(r.RemoteAddr),
					},
				},
				audit.PayloadKeyTargetResources: []audit.TargetResource{
					{
						TargetResourceName: utillog.Sanitize(r.URL.Path),
						TargetResourceType: auditTargetResourceType(r),
					},
				},
			})

			otelAuditMsg := audit.CreateOtelAuditMsg(log, r)
			otelAuditMsg.Record.CallerIdentities = map[msgs.CallerIdentityType][]msgs.CallerIdentityEntry{
				msgs.Username: {
					{
						Identity:    username,
						Description: "client username",
					},
				},
			}
			otelAuditMsg.Record.TargetResources = map[string][]msgs.TargetResourceEntry{
				auditTargetResourceType(r): {
					{
						Name: r.URL.Path,
					},
				},
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
				if statusCode >= http.StatusBadRequest {
					resultType = audit.ResultTypeFail
					otelAuditMsg.Record.OperationResult = msgs.Failure
					otelAuditMsg.Record.OperationResultDescription = fmt.Sprintf("Status code: %d", statusCode)
				}

				audit.EnsureDefaults(&otelAuditMsg.Record)
				if err := outelAuditClient.Send(r.Context(), otelAuditMsg); err != nil {
					log.Errorf("Portal - Error sending audit message: %v", err)
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

func auditTargetResourceType(r *http.Request) string {
	if matches := utillog.RXTolerantSubResourceID.FindStringSubmatch(r.URL.Path); matches != nil {
		return matches[len(matches)-1]
	}

	return ""
}
