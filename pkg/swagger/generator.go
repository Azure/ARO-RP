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
		exampleOpenShiftClusterPutParameter:            v20240812preview.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20240812preview.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterGetResponse:             v20240812preview.ExampleOpenShiftClusterGetResponse,
		exampleOpenShiftClusterPutOrPatchResponse:      v20240812preview.ExampleOpenShiftClusterPutOrPatchResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20240812preview.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20240812preview.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20240812preview.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOpenShiftVersionListResponse:            v20240812preview.ExampleOpenShiftVersionListResponse,
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
}

func New(api string) (*generator, error) {
	if val, ok := apis[api]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("api %s not found", api)
}
