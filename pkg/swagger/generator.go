package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/api/v20210131preview"
)

const apiv20200430Path = "github.com/Azure/ARO-RP/pkg/api/v20200430"
const apiv20210131previewPath = "github.com/Azure/ARO-RP/pkg/api/v20210131preview"

type generator struct {
	exampleOpenShiftClusterPutParameter            func() interface{}
	exampleOpenShiftClusterPatchParameter          func() interface{}
	exampleOpenShiftClusterResponse                func() interface{}
	exampleOpenShiftClusterCredentialsResponse     func() interface{}
	exampleOpenShiftClusterAdminKubeconfigResponse func() interface{}
	exampleOpenShiftClusterListResponse            func() interface{}
	exampleOperationListResponse                   func() interface{}

	systemData         bool
	kubeConfig         bool
	xmsEnum            []string
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
	apiv20210131previewPath: {
		exampleOpenShiftClusterPutParameter:            v20210131preview.ExampleOpenShiftClusterPutParameter,
		exampleOpenShiftClusterPatchParameter:          v20210131preview.ExampleOpenShiftClusterPatchParameter,
		exampleOpenShiftClusterResponse:                v20210131preview.ExampleOpenShiftClusterResponse,
		exampleOpenShiftClusterCredentialsResponse:     v20210131preview.ExampleOpenShiftClusterCredentialsResponse,
		exampleOpenShiftClusterListResponse:            v20210131preview.ExampleOpenShiftClusterListResponse,
		exampleOpenShiftClusterAdminKubeconfigResponse: v20210131preview.ExampleOpenShiftClusterAdminKubeconfigResponse,
		exampleOperationListResponse:                   api.ExampleOperationListResponse,

		xmsEnum:            []string{"VMSize", "SDNProvider"},
		commonTypesVersion: "v2",
		systemData:         true,
		kubeConfig:         true,
	},
}

func New(api string) (*generator, error) {
	if val, ok := apis[api]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("api %s not found", api)
}
