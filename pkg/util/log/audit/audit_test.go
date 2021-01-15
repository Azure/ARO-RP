package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestAudit(t *testing.T) {
	var (
		now          = time.Now().UTC()
		formattedNow = now.Format(time.RFC3339)
	)

	t.Run("can convert log entry to audit log", func(t *testing.T) {
		logger, h := test.NewNullLogger()
		auditLog := NewEntry(logger)
		auditLog.WithFields(logrus.Fields{
			EnvKeyEnvironment: azure.PublicCloud.Name,
			EnvKeyHostname:    "test-host",
			EnvKeyLocation:    "eastus",
			PayloadKeyCallerIdentities: []CallerIdentity{
				{
					CallerDisplayName:   "test-caller",
					CallerIdentityType:  CallerIdentityTypeSubscriptionID,
					CallerIdentityValue: "test-caller-value",
					CallerIPAddress:     "127.0.0.1",
				},
			},
			PayloadKeyCategory:      CategoryAuthorization,
			PayloadKeyOperationName: "initializeAuthorizers",
			PayloadKeyTargetResources: []TargetResource{
				{
					TargetResourceType: "test-resource-type",
					TargetResourceName: "test-resource",
				},
			},
			PayloadKeyResult: Result{
				ResultType:        ResultTypeSuccess,
				ResultDescription: "test-result-desc",
			},
			PayloadKeyRequestID: "request-id",
		}).Print("test message")

		assertAuditingOutput(t, h, []map[string]types.GomegaMatcher{
			{
				"level":         gomega.Equal(logrus.InfoLevel),
				"msg":           gomega.Equal("test message"),
				MetadataLogKind: gomega.Equal("ifxaudit"),
				MetadataCreatedTime: gomega.WithTransform(
					func(s string) time.Time {
						t, err := time.Parse(time.RFC3339, s)
						if err != nil {
							panic(err)
						}
						return t
					},
					gomega.BeTemporally("~", now, time.Second),
				),
				MetadataPayload: gomega.Equal(`{"env_ver":2.1,"env_name":"#Ifx.AuditSchema","env_time":"` + formattedNow + `","env_epoch":"` + epoch + `","env_seqNum":1,"env_flags":257,"env_appId":"","env_cloud_name":"AzurePublicCloud","env_cloud_role":"","env_cloud_roleInstance":"test-host","env_cloud_environment":"AzurePublicCloud","env_cloud_location":"eastus","env_cloud_ver":1,"CallerIdentities":[{"CallerDisplayName":"test-caller","CallerIdentityType":"SubscriptionID","CallerIdentityValue":"test-caller-value","CallerIpAddress":"127.0.0.1"}],"Category":"Authorization","OperationName":"initializeAuthorizers","Result":{"ResultType":"Success","ResultDescription":"test-result-desc"},"requestId":"request-id","TargetResources":[{"TargetResourceType":"test-resource-type","TargetResourceName":"test-resource"}]}`),
			},
		})
	})
}

// AssertAuditingOutput compares the audit logs in `h` with the expected entries
// in `expected`.
func assertAuditingOutput(t *testing.T, h *test.Hook, expected []map[string]types.GomegaMatcher) {
	assertAuditEntriesCount(t, h, expected)
	assertRequiredFields(t, h)
	assertMatchPayload(t, h, expected)
}

func assertAuditEntriesCount(t *testing.T, h *test.Hook, expected []map[string]types.GomegaMatcher) {
	if len(h.AllEntries()) != len(expected) {
		t.Fatalf("mismatch audit entries count. expected: %d, actual: %d",
			len(expected), len(h.AllEntries()))
	}
}

func assertRequiredFields(t *testing.T, h *test.Hook) {
	// required metadata
	var (
		missingMetadata  = []string{}
		requiredMetadata = []string{
			MetadataCreatedTime,
			MetadataPayload,
			MetadataLogKind,
		}
	)
	for _, e := range h.AllEntries() {
		for _, field := range requiredMetadata {
			if _, exists := e.Data[field]; !exists {
				missingMetadata = append(missingMetadata, field)
			}
		}
	}

	if len(missingMetadata) > 0 {
		t.Fatalf("missing required metadata: %s",
			strings.Join(missingMetadata, ","))
	}

	// required payload fields
	payloadErrors := []string{}
	for _, e := range h.AllEntries() {
		raw, ok := e.Data[MetadataPayload].(string)
		if !ok {
			payloadErrors = append(payloadErrors,
				"type assertion failed on audit payload")
			continue
		}

		var payload payload
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			payloadErrors = append(payloadErrors,
				fmt.Sprintf("malformed payload JSON: %s", err))
			continue
		}

		if payload.EnvVer == 0 {
			payloadErrors = append(payloadErrors,
				"EnvVer field must be non-zero")
		}

		if payload.EnvName == "" {
			payloadErrors = append(payloadErrors,
				"EnvName field must not be empty")
		}

		if payload.EnvTime == "" {
			payloadErrors = append(payloadErrors,
				"EnvTime field must not be empty")
		}

		if payload.EnvCloudName == "" {
			payloadErrors = append(payloadErrors,
				"EnvCloudName field must not be empty")
		}

		if payload.EnvCloudRoleInstance == "" {
			payloadErrors = append(payloadErrors,
				"EnvCloudRoleInstance field must not be empty")
		}

		if payload.EnvCloudLocation == "" {
			payloadErrors = append(payloadErrors,
				"EnvCloudLocation field must not be empty")
		}

		if payload.EnvCloudVer == 0 {
			payloadErrors = append(payloadErrors,
				"EnvCloudVer field must not be empty")
		}

		if len(payload.CallerIdentities) == 0 {
			payloadErrors = append(payloadErrors,
				"CallerIdentities field must not be empty")
		}

		if payload.Category == "" {
			payloadErrors = append(payloadErrors,
				"Category field must not be empty")
		}

		if payload.OperationName == "" {
			payloadErrors = append(payloadErrors,
				"OperationName field must not be empty")
		}

		if payload.Result.ResultType == "" {
			payloadErrors = append(payloadErrors,
				"Result field must not be empty")
		}

		if payload.RequestID == "" {
			payloadErrors = append(payloadErrors,
				"RequestID field must not be empty")
		}

		if len(payload.TargetResources) == 0 {
			payloadErrors = append(payloadErrors,
				"TargetResources field must not be empty")
		}
	}

	if len(payloadErrors) > 0 {
		t.Fatalf("missing required payload field: %s",
			strings.Join(payloadErrors, ","))
	}
}

func assertMatchPayload(t *testing.T, h *test.Hook, expected []map[string]types.GomegaMatcher) {
	var (
		entries = h.AllEntries()
		errors  = []string{}
	)

	for i, matchers := range expected {
		for field, matcher := range matchers {
			v := entries[i].Data[field]
			switch field {
			case "level":
				v = entries[i].Level
			case "msg":
				v = entries[i].Message
			}

			ok, err := matcher.Match(v)
			if err != nil {
				errors = append(errors, fmt.Sprintf("log %d, field %s, error %s", i, field, err))
			} else if !ok {
				errors = append(errors, fmt.Sprintf("log %d, field %s, %s", i, field, matcher.FailureMessage(v)))
			}
		}
	}

	if len(errors) > 0 {
		formatted := make([]string, 0, len(entries))

		for _, entry := range entries {
			b, _ := entry.Logger.Formatter.Format(entry)
			formatted = append(formatted, string(b))
		}

		t.Errorf("logging mismatch:\ngot:\n%s\nproblems:\n%s", strings.Join(formatted, ""), strings.Join(errors, "\n"))
	}
}
