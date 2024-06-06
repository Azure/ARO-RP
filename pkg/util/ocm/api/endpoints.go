package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"fmt"
	"text/template"
)

const (
	GetClusterListEndpointV1               = "/api/clusters_mgmt/v1/clusters"
	GetClusterUpgradePoliciesEndpointV1    = GetClusterListEndpointV1 + "/{{required \"ocmClusterID\"}}/upgrade_policies"
	CancelClusterUpgradePolicyEndpointV1   = GetClusterUpgradePoliciesEndpointV1 + "/{{required \"policyID\"}}/state"
	GetClusterUpgradePolicyStateEndpointV1 = GetClusterUpgradePoliciesEndpointV1 + "/{{required \"policyID\"}}/state"
)

func BuildEndpoint(templateStr string, params map[string]string) (string, error) {
	tmplFuncs := template.FuncMap{
		"required": func(key string) (string, error) {
			if value, ok := params[key]; ok {
				return value, nil
			}
			return "", fmt.Errorf("missing required parameter: %s", key)
		},
	}
	tmpl, err := template.New("endpoint").Funcs(tmplFuncs).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var endpoint bytes.Buffer
	if err := tmpl.Execute(&endpoint, params); err != nil {
		return "", err
	}

	return endpoint.String(), nil
}
