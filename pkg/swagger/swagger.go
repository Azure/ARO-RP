package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func Run(outputDir string) error {
	s := &Swagger{
		Swagger: "2.0",
		Info: &Info{
			Title:       "Azure Red Hat OpenShift Client",
			Description: "Rest API for Azure Red Hat OpenShift",
			Version:     "2019-12-31-preview",
		},
		Host:        "management.azure.com",
		Schemes:     []string{"https"},
		Consumes:    []string{"application/json"},
		Produces:    []string{"application/json"},
		Paths:       populateTopLevelPaths("Microsoft.RedHatOpenShift", "openShiftCluster", "OpenShift cluster"),
		Definitions: Definitions{},
		SecurityDefinitions: SecurityDefinitions{
			"azure_auth": {
				Type:             "oauth2",
				AuthorizationURL: "https://login.microsoftonline.com/common/oauth2/authorize",
				Flow:             "implicit",
				Description:      "Azure Active Directory OAuth2 Flow",
				Scopes: map[string]string{
					"user_impersonation": "impersonate your user account",
				},
			},
		},
		Security: []SecurityRequirement{
			{
				"azure_auth": []string{"user_impersonation"},
			},
		},
	}

	s.Paths["/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{resourceName}/listCredentials"] = &PathItem{
		Post: &Operation{
			Tags:        []string{"OpenShiftClusters"},
			Summary:     "Lists credentials of an OpenShift cluster with the specified subscription, resource group and resource name.",
			Description: "Lists credentials of an OpenShift cluster with the specified subscription, resource group and resource name.  The operation returns the credentials.",
			OperationID: "OpenShiftClusters_ListCredentials",
			Parameters:  populateParameters(3, "OpenShiftCluster", "OpenShift cluster"),
			Responses:   populateResponses("OpenShiftClusterCredentials", false, http.StatusOK),
		},
	}

	s.Paths["/providers/Microsoft.RedHatOpenShift/operations"] = &PathItem{
		Get: &Operation{
			Tags:        []string{"Operations"},
			Summary:     "Lists all of the available RP operations.",
			Description: "Lists all of the available RP operations.  The operation returns the RP operations.",
			OperationID: "Operations_List",
			Parameters:  populateParameters(0, "Operation", "Operation"),
			Responses:   populateResponses("OperationList", false, http.StatusOK),
		},
	}

	populateExamples(s.Paths)

	err := define(s.Definitions, "github.com/Azure/ARO-RP/pkg/api/v20191231preview", "OpenShiftClusterList", "OpenShiftClusterCredentials")
	if err != nil {
		return err
	}

	err = define(s.Definitions, "github.com/Azure/ARO-RP/pkg/api", "CloudError", "OperationList")
	if err != nil {
		return err
	}

	for _, azureResource := range []string{"OpenShiftCluster"} {
		s.Definitions[azureResource].AllOf = []Schema{
			{
				Ref: "../../../../../common-types/resource-management/v1/types.json#/definitions/TrackedResource",
			},
		}

		var properties []NameSchema

		for _, property := range s.Definitions[azureResource].Properties {
			if property.Name == "properties" {
				property.Schema.ClientFlatten = true
				properties = append(properties, property)
			}
		}

		s.Definitions[azureResource].Properties = properties
	}

	delete(s.Definitions, "Tags")

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	b = append(b, '\n')

	err = generateExamples(outputDir, s)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputDir+"/redhatopenshift.json", b, 0666)
}
