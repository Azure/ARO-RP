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
	v20230401 "github.com/Azure/ARO-RP/pkg/api/v20230401"
	v20230701preview "github.com/Azure/ARO-RP/pkg/api/v20230701preview"
	v20230904 "github.com/Azure/ARO-RP/pkg/api/v20230904"
	v20231122 "github.com/Azure/ARO-RP/pkg/api/v20231122"
	v20240812preview "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
)

const apiv20200430Path = "github.com/Azure/ARO-RP/pkg/api/v20200430"
const apiv20210901previewPath = "github.com/Azure/ARO-RP/pkg/api/v20210901preview"
const apiv20220401Path = "github.com/Azure/ARO-RP/pkg/api/v20220401"
const apiv20220904Path = "github.com/Azure/ARO-RP/pkg/api/v20220904"
const apiv20230401Path = "github.com/Azure/ARO-RP/pkg/api/v20230401"
const apiv20230701previewPath = "github.com/Azure/ARO-RP/pkg/api/v20230701preview"
const apiv20230904Path = "github.com/Azure/ARO-RP/pkg/api/v20230904"
const apiv20231122Path = "github.com/Azure/ARO-RP/pkg/api/v20231122"
const apiv20240812previewPath = "github.com/Azure/ARO-RP/pkg/api/v20240812preview"

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
	exampleOpenShiftClusterGetResponse             func() interface{}
	exampleOpenShiftClusterPutOrPatchResponse      func() interface{}
	exampleOpenShiftClusterCredentialsResponse     func() interface{}
	exampleOpenShiftClusterAdminKubeconfigResponse func() interface{}
	exampleOpenShiftClusterListResponse            func() interface{}
	exampleOpenShiftVersionListResponse            func() interface{}
	exampleOperationListResponse                   func() interface{}

	systemData           bool
	kubeConfig           bool
	installVersionList   bool
	clusterManager       bool
	workerProfilesStatus bool
	xmsEnum              []string
	xmsSecretList        []string
	xmsIdentifiers       []string
	commonTypesVersion   string
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
	apiv20230401Path: {
		exampleSyncSetPutParameter:                     v20230401.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20230401.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20230401.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20230401.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20230401.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20230401.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20230401.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20230401.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20230401.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20230401.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20230401.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20230401.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20230401.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20230401.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20230401.ExampleSecretResponse,
		exampleSecretListResponse:                      v20230401.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20230401.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20230401.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20230401.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20230401.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20230401.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20230401.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20230401.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility", "OutboundType", "PreconfiguredNSG"},
		xmsSecretList:      []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:     []string{},
		commonTypesVersion: "v3",
		systemData:         true,
		clusterManager:     true,
		installVersionList: true,
		kubeConfig:         true,
	},
	apiv20230701previewPath: {
		exampleSyncSetPutParameter:                     v20230701preview.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20230701preview.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20230701preview.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20230701preview.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20230701preview.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20230701preview.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20230701preview.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20230701preview.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20230701preview.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20230701preview.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20230701preview.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20230701preview.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20230701preview.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20230701preview.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20230701preview.ExampleSecretResponse,
		exampleSecretListResponse:                      v20230701preview.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20230701preview.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20230701preview.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20230701preview.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20230701preview.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20230701preview.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20230701preview.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20230701preview.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility", "OutboundType"},
		xmsSecretList:      []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:     []string{},
		commonTypesVersion: "v3",
		systemData:         true,
		clusterManager:     true,
		installVersionList: true,
		kubeConfig:         true,
	},
	apiv20230904Path: {
		exampleSyncSetPutParameter:                     v20230904.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20230904.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20230904.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20230904.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20230904.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20230904.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20230904.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20230904.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20230904.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20230904.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20230904.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20230904.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20230904.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20230904.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20230904.ExampleSecretResponse,
		exampleSecretListResponse:                      v20230904.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20230904.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20230904.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterGetResponse:             v20230904.ExampleOpenShiftClusterGetResponse,
		exampleOpenShiftClusterPutOrPatchResponse:      v20230904.ExampleOpenShiftClusterPutOrPatchResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20230904.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20230904.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20230904.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20230904.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:              []string{"EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility", "OutboundType"},
		xmsSecretList:        []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:       []string{},
		commonTypesVersion:   "v3",
		systemData:           true,
		clusterManager:       true,
		installVersionList:   true,
		kubeConfig:           true,
		workerProfilesStatus: true,
	},
	apiv20231122Path: {
		exampleSyncSetPutParameter:                     v20231122.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20231122.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20231122.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20231122.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20231122.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20231122.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20231122.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20231122.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20231122.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20231122.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20231122.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20231122.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20231122.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20231122.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20231122.ExampleSecretResponse,
		exampleSecretListResponse:                      v20231122.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20231122.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20231122.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterGetResponse:             v20231122.ExampleOpenShiftClusterGetResponse,
		exampleOpenShiftClusterPutOrPatchResponse:      v20231122.ExampleOpenShiftClusterPutOrPatchResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20231122.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20231122.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20231122.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20231122.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:              []string{"ProvisioningState", "PreconfiguredNSG", "EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility", "OutboundType"},
		xmsSecretList:        []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:       []string{},
		commonTypesVersion:   "v3",
		systemData:           true,
		clusterManager:       true,
		installVersionList:   true,
		kubeConfig:           true,
		workerProfilesStatus: true,
	},
	apiv20240812previewPath: {
		exampleSyncSetPutParameter:                     v20240812preview.ExampleSyncSetPutParameter,
		exampleSyncSetPatchParameter:                   v20240812preview.ExampleSyncSetPatchParameter,
		exampleSyncSetResponse:                         v20240812preview.ExampleSyncSetResponse,
		exampleSyncSetListResponse:                     v20240812preview.ExampleSyncSetListResponse,
		exampleMachinePoolPutParameter:                 v20240812preview.ExampleMachinePoolPutParameter,
		exampleMachinePoolPatchParameter:               v20240812preview.ExampleMachinePoolPatchParameter,
		exampleMachinePoolResponse:                     v20240812preview.ExampleMachinePoolResponse,
		exampleMachinePoolListResponse:                 v20240812preview.ExampleMachinePoolListResponse,
		exampleSyncIdentityProviderPutParameter:        v20240812preview.ExampleSyncIdentityProviderPutParameter,
		exampleSyncIdentityProviderPatchParameter:      v20240812preview.ExampleSyncIdentityProviderPatchParameter,
		exampleSyncIdentityProviderResponse:            v20240812preview.ExampleSyncIdentityProviderResponse,
		exampleSyncIdentityProviderListResponse:        v20240812preview.ExampleSyncIdentityProviderListResponse,
		exampleSecretPutParameter:                      v20240812preview.ExampleSecretPutParameter,
		exampleSecretPatchParameter:                    v20240812preview.ExampleSecretPatchParameter,
		exampleSecretResponse:                          v20240812preview.ExampleSecretResponse,
		exampleSecretListResponse:                      v20240812preview.ExampleSecretListResponse,
		exampleOpenShiftClusterPutParameter:            v20240812preview.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20240812preview.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterGetResponse:             v20240812preview.ExampleOpenShiftClusterGetResponse,
		exampleOpenShiftClusterPutOrPatchResponse:      v20240812preview.ExampleOpenShiftClusterPutOrPatchResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20240812preview.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20240812preview.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20240812preview.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20240812preview.ExampleOpenShiftVersionListResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:              []string{"ProvisioningState", "PreconfiguredNSG", "EncryptionAtHost", "FipsValidatedModules", "SoftwareDefinedNetwork", "Visibility", "OutboundType", "ResourceIdentityType"},
		xmsSecretList:        []string{"kubeconfig", "kubeadminPassword", "secretResources"},
		xmsIdentifiers:       []string{},
		commonTypesVersion:   "v3",
		systemData:           true,
		clusterManager:       true,
		installVersionList:   true,
		kubeConfig:           true,
		workerProfilesStatus: true,
	},
}

func New(api string) (*generator, error) {
	if val, ok := apis[api]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("api %s not found", api)
}
