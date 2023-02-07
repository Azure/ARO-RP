package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/api/v20210901preview"
	v20220401 "github.com/Azure/ARO-RP/pkg/api/v20220401"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/api/v20230610preview"
)

const apiv20200430Path = "github.com/Azure/ARO-RP/pkg/api/v20200430"
const apiv20210901previewPath = "github.com/Azure/ARO-RP/pkg/api/v20210901preview"
const apiv20220401Path = "github.com/Azure/ARO-RP/pkg/api/v20220401"
const apiv20220904Path = "github.com/Azure/ARO-RP/pkg/api/v20220904"
const apiv20230610previewPath = "github.com/Azure/ARO-RP/pkg/api/v20230610preview"

type generator struct {
	exampleSyncSetPutParameter                     func() interface{}
	exampleSyncSetPatchParameter                   func() interface{}
	exampleSyncSetResponse                         func() interface{}
	exampleSyncSetListResponse                     func() interface{}
	exampleMachinePoolPutParameter                 func() interface{}
	exampleMachinePoolPatchParameter               func() interface{}
	exampleMachinePoolResponse                     func() interface{}
	exampleMachinePoolListResponse                 func() interface{}
	exampleSyncIdentityProviderPutParameter        func() interface{}
	exampleSyncIdentityProviderPatchParameter      func() interface{}
	exampleSyncIdentityProviderResponse            func() interface{}
	exampleSyncIdentityProviderListResponse        func() interface{}
	exampleSecretPutParameter                      func() interface{}
	exampleSecretPatchParameter                    func() interface{}
	exampleSecretResponse                          func() interface{}
	exampleSecretListResponse                      func() interface{}
	exampleOpenShiftClusterPutParameter            func() interface{}
	exampleOpenShiftClusterPatchParameter          func() interface{}
	exampleOpenShiftClusterResponse                func() interface{}
	exampleOpenShiftClusterCredentialsResponse     func() interface{}
	exampleOpenShiftClusterAdminKubeconfigResponse func() interface{}
	exampleOpenShiftClusterListResponse            func() interface{}
	exampleOpenShiftVersionListResponse            func() interface{}
	exampleOperationListResponse                   func() interface{}

	systemData         bool
	kubeConfig         bool
	installVersionList bool
	clusterManager     bool
	xmsEnum            []string
	xmsSecretList      []string
	xmsIdentifiers     []string
	commonTypesVersion string
}

var apis = map[string]*generator{
	apiv20200430Path: {
		exampleOpenShiftClusterPutParameter:        v20200430.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:      v20200430.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:            v20200430.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse: v20200430.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:        v20200430.ExampleOpenShiftClusterListResponse,
		exampleOperationListResponse:               api.ExampleOperationListResponse,

		commonTypesVersion: "v1",
		xmsEnum:            []string{},
	},
	apiv20210901previewPath: {
		exampleOpenShiftClusterPutParameter:            v20210901preview.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20210901preview.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20210901preview.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20210901preview.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20210901preview.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20210901preview.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"VMSize", "SoftwareDefinedNetwork", "EncryptionAtHost", "Visibility"},
		xmsSecretList:      []string{"kubeconfig", "kubeadminPassword"},
		commonTypesVersion: "v2",
		systemData:         true,
		kubeConfig:         true,
	},
	apiv20220401Path: {
		exampleOpenShiftClusterPutParameter:            v20220401.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20220401.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20220401.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20220401.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20220401.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20220401.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility"},
		xmsSecretList:      []string{"kubeconfig", "kubeadminPassword"},
		xmsIdentifiers:     []string{},
		commonTypesVersion: "v2",
		systemData:         true,
		kubeConfig:         true,
	},
	apiv20220904Path: {
		exampleSyncSetPutParameter:                     v20220904.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20220904.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20220904.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20220904.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20220904.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20220904.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20220904.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20220904.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20220904.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20220904.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20220904.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20220904.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20220904.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20220904.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20220904.ExampleSecretResponse,
		exampleSecretListResponse:                      v20220904.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20220904.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20220904.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20220904.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20220904.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20220904.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20220904.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20220904.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility"},
		xmsSecretList:      []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:     []string{},
		commonTypesVersion: "v3",
		systemData:         true,
		clusterManager:     true,
		installVersionList: true,
		kubeConfig:         true,
	},
	apiv20230610previewPath: {
		exampleSyncSetPutParameter:                     v20230610preview.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20230610preview.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20230610preview.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20230610preview.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20230610preview.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20230610preview.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20230610preview.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20230610preview.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20230610preview.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20230610preview.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20230610preview.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20230610preview.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20230610preview.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20230610preview.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20230610preview.ExampleSecretResponse,
		exampleSecretListResponse:                      v20230610preview.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20230610preview.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20230610preview.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20230610preview.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20230610preview.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20230610preview.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20230610preview.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20230610preview.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility"},
		xmsSecretList:      []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:     []string{},
		commonTypesVersion: "v3",
		systemData:         true,
		clusterManager:     true,
		installVersionList: true,
		kubeConfig:         true,
	},
}

func New(api string) (*generator, error) {
	if val, ok := apis[api]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("api %s not found", api)
}
