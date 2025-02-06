package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	// additionalLogFieldsAnnotation is an annotation on a ClusterDeployment
	// object which must contain a marshalled json object with additional
	// log fields. We use it to make sure that Hive and installer add fields
	// such as resource_id and correlation_id and into their logs.
	// If the annotation is set, Hive will be adding these fields "as is" into
	// its logs every time it logs something in relation to a
	// given ClusterDeployment. Hive will also be passing these fields into
	// the installer for cluster provisioning and deprovisioning.
	// If the annotation is set, Hive will add component="hive"
	// field into its logs and make installer add component="installer"
	// into installer logs.
	additionalLogFieldsAnnotation = "hive.openshift.io/additional-log-fields"
)

// EnrichHiveWithCorrelationData sets correlation log fields based on correlationData struct.
func EnrichHiveWithCorrelationData(cd *hivev1.ClusterDeployment, correlationData *api.CorrelationData) error {
	if correlationData == nil {
		return nil
	}

	return patchClusterDeploymentLogFields(cd, func(logFields map[string]string) {
		logFields["correlation_id"] = correlationData.CorrelationID
		logFields["client_request_id"] = correlationData.ClientRequestID
		logFields["request_id"] = correlationData.RequestID
		logFields["client_principal_name"] = correlationData.ClientPrincipalName
	})
}

// ResetHiveCorrelationData removes correlation log fields from ClusterDeployment, if present.
func ResetHiveCorrelationData(cd *hivev1.ClusterDeployment) error {
	return patchClusterDeploymentLogFields(cd, func(logFields map[string]string) {
		delete(logFields, "correlation_id")
		delete(logFields, "client_request_id")
		delete(logFields, "request_id")
		delete(logFields, "client_principal_name")
	})
}

// EnrichHiveWithResourceID sets resource log fields based on a cluster resourceID.
func EnrichHiveWithResourceID(cd *hivev1.ClusterDeployment, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	return patchClusterDeploymentLogFields(cd, func(logFields map[string]string) {
		logFields["resource_id"] = strings.ToLower(resourceID)
		logFields["subscription_id"] = strings.ToLower(r.SubscriptionID)
		logFields["resource_group"] = strings.ToLower(r.ResourceGroup)
		logFields["resource_name"] = strings.ToLower(r.ResourceName)
	})
}

func patchClusterDeploymentLogFields(cd *hivev1.ClusterDeployment, mutator func(fields map[string]string)) error {
	logFields := map[string]string{}
	if val := cd.Annotations[additionalLogFieldsAnnotation]; val != "" {
		err := json.Unmarshal([]byte(val), &logFields)
		if err != nil {
			return err
		}
	}

	mutator(logFields)

	b, err := json.Marshal(logFields)
	if err != nil {
		return err
	}

	if cd.Annotations == nil {
		cd.Annotations = make(map[string]string)
	}
	cd.Annotations[additionalLogFieldsAnnotation] = string(b)
	return nil
}
