package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	cloudcredentialv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
)

type permissionSet struct {
	name             string
	manifest         string
	roleDefinitionId string
}

var (
	version        = "4.15.30"
	tempDir        = path.Join("/tmp/credreqs", version)
	permissionSets = []{
		{
			"Cloud Controller Manager",
			"0000_26_cloud-controller-manager-operator_14_credentialsrequest-azure.yaml",
			"a1f96423-95ce-4224-ab27-4e3dc72facd4",
		},
		{
			"Machine API Operator",
			"0000_30_machine-api-operator_00_credentials-request.yaml",
			"0358943c-7e01-48ba-8889-02cc51d78637",
		},
		{
			"Cluster Image Registry Operator",
			"0000_50_cluster-image-registry-operator_01-registry-credentials-request-azure.yaml",
			"8b32b316-c2f5-4ddf-b05b-83dacd2d08b5",
		},
		{
			"Cluster Ingress Operator",
			"0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml",
			"0336e1d3-7a87-462b-b6db-342b63f7802c",
		},
		{
			"Cluster Network Operator",
			"0000_50_cluster-network-operator_02-cncc-credentials.yaml",
			"be7a6435-15ae-4171-8f30-4a343eff9e8f",
		},
		{
			"Cluster Storage Operator",
			"0000_50_cluster-storage-operator_03_credentials_request_azure.yaml",
			"5b7237c5-45e1-49d6-bc18-a1f62f400748",
		},
		{
			"Cluster Storage Operator (Azure File)",
			"0000_50_cluster-storage-operator_03_credentials_request_azure_file.yaml",
			"0d7aedc0-15fd-4a67-a412-efad370c947e",
		},
	}
)

type manager struct {
	roleDefinitions armauthorization.RoleDefinitionsClient
}

func main() {
	// oc adm release extract --credentials-requests --to /tmp/credreqs quay.io/openshift-release-dev/ocp-release:4.15.30-x86_64
	for _, ps := range permissionSets {
	}
}

func (m *manager) checkPermissions(ctx context.Context, ps *permissionSets) (actions, dataActions []string, err error) {
	// Get the AzureProviderSpec from the credentials request
	f, err := os.Open(path.Join(tempDir, ps.manifest))
	credreq, err := GetAzureCredentialsRequest(f)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Azure credentials request: %w", err)
	}
	var spec cloudcredentialv1.AzureProviderSpec
	err = cloudcredentialv1.Codec.DecodeProviderSpec(credreq.Spec.ProviderSpec, &spec)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode AzureProviderSpec: %w", err)
	}

	// Get the role definition
	resp, err := m.roleDefinitions.GetByID(ctx, ps.roleDefinitionId, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get role definition: %w", err)
	}
	roleDef := resp.RoleDefinition

	// It's not guaranteed to have only one permission, but it's a good enough assumption for now.
	// Expect here to make sure the test fails if the assumption is wrong.
	if len(roleDef.Properties.Permissions) != 1 {
		return nil, nil, fmt.Errorf("role definition has %d permissions, expected 1", len(roleDef.Properties.Permissions))
	}
	permission := roleDef.Properties.Permissions[0]

	return missingActions(&spec, permission), missingDataActions(&spec, permission), nil
}

func missingActions(credreq *cloudcredentialv1.AzureProviderSpec, permission *sdkauthorization.Permission) []string {
	var result []string
	for _, action := range credreq.Permissions {
		if !slices.ContainsFunc(permission.Actions, func(a *string) bool { return *a == action }) {
			result = append(result, action)
		}
	}
	return result
}

func missingDataActions(credreq *cloudcredentialv1.AzureProviderSpec, permission *sdkauthorization.Permission) []string {
	var result []string
	for _, action := range credreq.DataPermissions {
		if !slices.ContainsFunc(permission.DataActions, func(a *string) bool { return *a == action }) {
			result = append(result, action)
		}
	}
	return result
}

func GetAzureCredentialsRequest(r io.Reader) (*cloudcredentialv1.CredentialsRequest, error) {
	dec := yaml.NewYAMLOrJSONDecoder(r, 4096)
	var credreq cloudcredentialv1.CredentialsRequest
	for {
		err := dec.Decode(&credreq)
		if errors.Is(err, io.EOF) {
			return nil, errors.New("azure credentials request not found")
		}
		if err != nil {
			return nil, err
		}
		unknown := runtime.Unknown{}
		err = cloudcredentialv1.Codec.DecodeProviderSpec(credreq.Spec.ProviderSpec, &unknown)
		if err != nil {
			return nil, err
		}
		if unknown.Kind == "AzureProviderSpec" {
			return &credreq, nil
		}
	}
}
