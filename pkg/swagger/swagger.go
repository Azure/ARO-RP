package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// resourceName represents the tracked resource, OpenShiftCluster
var resourceName = "OpenShiftCluster"

// proxyResources represent the list of proxy resources - these are resources with operations, but do not exist in the Azure Portal
// https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/proxy-api-reference.md
var proxyResources = []string{
	"SyncSet",
	"SyncIdentityProvider",
	"MachinePool",
	"Secret",
}

func Run(api, outputDir string) error {
	g, err := New(api)
	if err != nil {
		return err
	}

	s := &Swagger{
		Swagger: "2.0",
		Info: &Info{
			Title:       "Azure Red Hat OpenShift Client",
			Description: "Rest API for Azure Red Hat OpenShift 4",
			Version:     stringutils.LastTokenByte(outputDir, '/'),
		},
		Host:        "management.azure.com",
		Schemes:     []string{"https"},
		Consumes:    []string{"application/json"},
		Produces:    []string{"application/json"},
		Paths:       g.populateTopLevelPaths("Microsoft.RedHatOpenShift", "openShiftCluster", "OpenShift cluster"),
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

	s.Paths["/providers/Microsoft.RedHatOpenShift/operations"] = &PathItem{
		Get: &Operation{
			Tags:        []string{"Operations"},
			Summary:     "Lists all of the available RP operations.",
			Description: "The operation returns the RP operations.",
			OperationID: "Operations_List",
			Parameters:  g.populateParameters(0, "Operation", "Operation"),
			Responses:   g.populateResponses("OperationList", false, http.StatusOK),
			Pageable: &Pageable{
				NextLinkName: "nextLink",
			},
		},
	}

	s.Paths["/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{resourceName}/listCredentials"] = &PathItem{
		Post: &Operation{
			Tags:        []string{"OpenShiftClusters"},
			Summary:     "Lists credentials of an OpenShift cluster with the specified subscription, resource group and resource name.",
			Description: "The operation returns the credentials.",
			OperationID: "OpenShiftClusters_ListCredentials",
			Parameters:  g.populateParameters(3, "OpenShiftCluster", "OpenShift cluster"),
			Responses:   g.populateResponses("OpenShiftClusterCredentials", false, http.StatusOK),
		},
	}

	if g.kubeConfig {
		s.Paths["/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{resourceName}/listAdminCredentials"] = &PathItem{
			Post: &Operation{
				Tags:        []string{"OpenShiftClusters"},
				Summary:     "Lists admin kubeconfig of an OpenShift cluster with the specified subscription, resource group and resource name.",
				Description: "The operation returns the admin kubeconfig.",
				OperationID: "OpenShiftClusters_ListAdminCredentials",
				Parameters:  g.populateParameters(3, "OpenShiftCluster", "OpenShift cluster"),
				Responses:   g.populateResponses("OpenShiftClusterAdminKubeconfig", false, http.StatusOK),
			},
		}
	}

	if g.installVersionList {
		s.Paths["/subscriptions/{subscriptionId}/providers/Microsoft.RedHatOpenShift/locations/{location}/listInstallVersions"] = &PathItem{
			Get: &Operation{
				Tags:        []string{"InstallVersions"},
				Summary:     "Lists all OpenShift versions available to install in the specified location.",
				Description: "The operation returns the installable OpenShift versions as strings.",
				OperationID: "InstallVersions_List",
				Parameters:  g.populateParameters(6, "InstallVersion", "Install Versions"),
				Responses:   g.populateResponses("InstallVersion", false, http.StatusOK),
			},
		}
	}

	if g.clusterManager {
		g.populateChildResourcePaths(s.Paths, "Microsoft.RedHatOpenShift", "openShiftCluster", "syncSet", "SyncSet")
		g.populateChildResourcePaths(s.Paths, "Microsoft.RedHatOpenShift", "openShiftCluster", "machinePool", "MachinePool")
		g.populateChildResourcePaths(s.Paths, "Microsoft.RedHatOpenShift", "openShiftCluster", "syncIdentityProvider", "SyncIdentityProvider")
		g.populateChildResourcePaths(s.Paths, "Microsoft.RedHatOpenShift", "openShiftCluster", "secret", "Secret")
	}
	populateExamples(s.Paths)

	// This begins to define definitions required for the paths, parameters, and responses defined above
	names := []string{"OpenShiftClusterList", "OpenShiftClusterCredentials"}
	if g.kubeConfig {
		names = append(names, "OpenShiftClusterAdminKubeconfig")
	}

	if g.installVersionList {
		names = append(names, "InstallVersionList")
	}

	if g.clusterManager {
		// This needs to be the top level struct
		// in most cases, the "list" struct (a collection of resources)
		names = append(names, "SyncSetList", "MachinePoolList", "SyncIdentityProviderList", "SecretList")
	}

	err = define(s.Definitions, api, g.xmsEnum, g.xmsSecretList, g.xmsIdentifiers, names...)
	if err != nil {
		return err
	}

	names = []string{"CloudError", "OperationList"}
	err = define(s.Definitions, "github.com/Azure/ARO-RP/pkg/api", g.xmsEnum, g.xmsSecretList, g.xmsIdentifiers, names...)
	if err != nil {
		return err
	}

	// This begins the ARM / Azure Resources definition generation
	azureResources := []string{
		resourceName,
	}

	if g.clusterManager {
		azureResources = append(azureResources, "SyncSet", "MachinePool", "SyncIdentityProvider", "Secret")
	}

	for _, azureResource := range azureResources {
		def, err := deepCopy(s.Definitions[azureResource])
		if err != nil {
			return err
		}
		update := def.(*Schema)

		var properties []NameSchema

		for _, property := range s.Definitions[azureResource].Properties {
			switch property.Name {
			case "id", "name", "type", "location":
			case "properties":
				property.Schema.ClientFlatten = true
				fallthrough
			default:
				properties = append(properties, property)
			}
		}

		update.Properties = properties
		s.Definitions[azureResource+"Update"] = update

		// If this resource is not a proxy resource mark as tracked resource
		// otherwise, its a proxy resource and we remove the proxyResource .Allof ref for the Update definition
		// in order to make azure-rest-specs-api validation happy
		if !contains(proxyResources, azureResource) {
			s.Definitions[azureResource].AllOf = []Schema{
				{
					Ref: "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/definitions/TrackedResource",
				},
			}
		} else {
			if def, ok := s.Definitions[azureResource+"Update"]; ok {
				def.AllOf = []Schema{}
			}
		}

		properties = nil
		for _, property := range s.Definitions[azureResource].Properties {
			if property.Name == "properties" {
				property.Schema.ClientFlatten = true
				properties = append(properties, property)
			}
		}
		s.Definitions[azureResource].Properties = properties

		if g.systemData {
			s.defineSystemData([]string{azureResource, azureResource + "Update"}, g.commonTypesVersion)
		}
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	b = append(b, '\n')

	err = g.generateExamples(outputDir, s)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputDir+"/redhatopenshift.json", b, 0666)
}

func deepCopy(v interface{}) (interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	w := reflect.New(reflect.TypeOf(v)).Interface()
	err = json.Unmarshal(b, w)
	if err != nil {
		return nil, err
	}

	return reflect.ValueOf(w).Elem().Interface(), nil
}

// defineSystemData will configure systemData fields for required definitions.
// SystemData is not user consumable, so we remove definitions from auto-generated code
// In addition to this we use common-types definition so we replace one we generate with common-types
func (s *Swagger) defineSystemData(resources []string, commonVersion string) {
	for _, resource := range resources {
		s.Definitions[resource].Properties = removeNamedSchemas(s.Definitions[resource].Properties, "systemData")

		// SystemData is not user side consumable type. It is being returned as Read-Only,
		// but should not be generated into API or swagger as API/SDK type
		delete(s.Definitions, "SystemData")
		delete(s.Definitions, "CreatedByType")
		s.Definitions[resource].Properties = append(s.Definitions[resource].Properties,
			NameSchema{
				Name: "systemData",
				Schema: &Schema{
					ReadOnly:    true,
					Description: "The system meta data relating to this resource.",
					Ref:         "../../../../../common-types/resource-management/" + commonVersion + "/types.json#/definitions/systemData",
				},
			})
	}
}

func removeNamedSchemas(list NameSchemas, remove string) NameSchemas {
	var result NameSchemas
	for _, schema := range list {
		if schema.Name == remove {
			continue
		}
		result = append(result, schema)
	}

	return result
}

// TODO: once we upgrade to go 1.18 we can use the slices.Contains function
// until then, heres this helper func to check if the slice exists after we put it in a map
func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}
