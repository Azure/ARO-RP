package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
)

const (
	authenticationTypeMetricsTopic = "cluster.cloudCredentialsMode"
)

func (mon *Monitor) emitClusterAuthenticationType(ctx context.Context) error {
	cloudCredentialObject, err := mon.operatorcli.OperatorV1().CloudCredentials().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		mon.log.Errorf("Error in getting the cluster authentication type: %v", err)
		return err
	}

	if cloudCredentialObject.Spec.CredentialsMode == operatorv1.CloudCredentialsModeManual {
		mon.emitGauge(authenticationTypeMetricsTopic, 1, map[string]string{
			"type": "managedIdentity",
		})
	}

	if cloudCredentialObject.Spec.CredentialsMode == operatorv1.CloudCredentialsModeDefault {
		mon.emitGauge(authenticationTypeMetricsTopic, 1, map[string]string{
			"type": "servicePrincipal",
		})
	}

	return nil
}
