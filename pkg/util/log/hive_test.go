package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEnrichHiveWithCorrelationData(t *testing.T) {
	for _, tt := range []struct {
		name            string
		correlationData *api.CorrelationData
		existingFields  bool
		wantLogFields   map[string]string
	}{
		{
			name: "correlation data provided, no existing fields on ClusterDeployment",
			correlationData: &api.CorrelationData{
				CorrelationID:       "fake-correlation-id",
				ClientRequestID:     "fake-client-request-id",
				RequestID:           "fake-request-id",
				ClientPrincipalName: "fake-client-principal-name",
				RequestTime:         time.Now(),
			},
			wantLogFields: map[string]string{
				"correlation_id":        "fake-correlation-id",
				"client_request_id":     "fake-client-request-id",
				"request_id":            "fake-request-id",
				"client_principal_name": "fake-client-principal-name",
			},
		},
		{
			name:           "correlation data provided, existing fields are on ClusterDeployment - override",
			existingFields: true,
			correlationData: &api.CorrelationData{
				CorrelationID:       "fake-correlation-id",
				ClientRequestID:     "fake-client-request-id",
				RequestID:           "fake-request-id",
				ClientPrincipalName: "fake-client-principal-name",
				RequestTime:         time.Now(),
			},
			wantLogFields: map[string]string{
				"correlation_id":        "fake-correlation-id",
				"client_request_id":     "fake-client-request-id",
				"request_id":            "fake-request-id",
				"client_principal_name": "fake-client-principal-name",
				"fake_existing_field":   "existing-fake-value",
			},
		},
		{
			name:           "correlation data was not provided, existing fields are on ClusterDeployment",
			existingFields: true,
			wantLogFields: map[string]string{
				"correlation_id":      "existing-fake-correlation-id",
				"fake_existing_field": "existing-fake-value",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cd := &hivev1.ClusterDeployment{}
			if tt.existingFields {
				cd.Annotations = map[string]string{
					additionalLogFieldsAnnotation: `{
						"fake_existing_field": "existing-fake-value",
						"correlation_id": "existing-fake-correlation-id"
					}`,
				}
			}

			err := EnrichHiveWithCorrelationData(cd, tt.correlationData)
			if err != nil {
				t.Error(err)
			}

			var actualLogFields map[string]string
			if val := cd.Annotations[additionalLogFieldsAnnotation]; val != "" {
				err := json.Unmarshal([]byte(val), &actualLogFields)
				if err != nil {
					t.Fatal(err)
				}
			}

			if !reflect.DeepEqual(actualLogFields, tt.wantLogFields) {
				t.Error(cmp.Diff(actualLogFields, tt.wantLogFields))
			}
		})
	}
}

func TestResetHiveCorrelationData(t *testing.T) {
	for _, tt := range []struct {
		name           string
		existingFields bool
		wantLogFields  map[string]string
		wantErr        string
	}{
		{
			name:           "removes existing fields",
			existingFields: true,
			wantLogFields: map[string]string{
				"fake_field_1": "existing-fake-field-1-value",
			},
		},
		{
			name:          "removes non-existing fields",
			wantLogFields: map[string]string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cd := &hivev1.ClusterDeployment{}
			if tt.existingFields {
				cd.Annotations = map[string]string{
					additionalLogFieldsAnnotation: `{
						"correlation_id":        "existing-fake-correlation-id",
						"client_request_id":     "existing-fake-client-request-id",
						"request_id":            "existing-fake-request-id",
						"client_principal_name": "existing-fake-client-principal-name",
						"fake_field_1":          "existing-fake-field-1-value"
					}`,
				}
			}

			err := ResetHiveCorrelationData(cd)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			var actualLogFields map[string]string
			if val := cd.Annotations[additionalLogFieldsAnnotation]; val != "" {
				err := json.Unmarshal([]byte(val), &actualLogFields)
				if err != nil {
					t.Fatal(err)
				}
			}

			if !reflect.DeepEqual(actualLogFields, tt.wantLogFields) {
				t.Error(cmp.Diff(actualLogFields, tt.wantLogFields))
			}
		})
	}
}

func TestEnrichHiveWithResourceID(t *testing.T) {
	fakeResourceIDWithMixedCaps := "/subscriptions/0000000F-000f-0000-0000-000000000000/resoUrCegroups/resOurceGroup/providers/microsoft.redhatopenshift/openshiftclusters/rESourceName"

	for _, tt := range []struct {
		name           string
		resourceID     string
		existingFields bool
		wantLogFields  map[string]string
		wantErr        string
	}{
		{
			name:       "resource id provided, no existing fields on ClusterDeployment",
			resourceID: fakeResourceIDWithMixedCaps,
			wantLogFields: map[string]string{
				"resource_group":  "resourcegroup",
				"resource_id":     "/subscriptions/0000000f-000f-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
				"resource_name":   "resourcename",
				"subscription_id": "0000000f-000f-0000-0000-000000000000",
			},
		},
		{
			name:           "resource id provided, existing fields are on ClusterDeployment - override",
			existingFields: true,
			resourceID:     fakeResourceIDWithMixedCaps,
			wantLogFields: map[string]string{
				"resource_group":      "resourcegroup",
				"resource_id":         "/subscriptions/0000000f-000f-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
				"resource_name":       "resourcename",
				"subscription_id":     "0000000f-000f-0000-0000-000000000000",
				"fake_existing_field": "existing-fake-value",
			},
		},
		{
			name:    "invalid resource id was provided",
			wantErr: "parsing failed for . Invalid resource Id format",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cd := &hivev1.ClusterDeployment{}
			if tt.existingFields {
				cd.Annotations = map[string]string{
					additionalLogFieldsAnnotation: `{
						"fake_existing_field": "existing-fake-value",
						"resource_id":         "existing-fake-resource-id"
					}`,
				}
			}

			err := EnrichHiveWithResourceID(cd, tt.resourceID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			var actualLogFields map[string]string
			if val := cd.Annotations[additionalLogFieldsAnnotation]; val != "" {
				err := json.Unmarshal([]byte(val), &actualLogFields)
				if err != nil {
					t.Fatal(err)
				}
			}

			if !reflect.DeepEqual(actualLogFields, tt.wantLogFields) {
				t.Error(cmp.Diff(actualLogFields, tt.wantLogFields))
			}
		})
	}
}
