package swagger

import (
	"encoding/json"
	"net/http"
	"os"
)

func Run(outputFile, shortVersion string) error {
	longVersion, err := longVersion(shortVersion)
	if err != nil {
		return err
	}
	s := &Swagger{
		Swagger: "2.0",
		Info: &Info{
			Title:       "Azure Red Hat OpenShift",
			Description: "Rest API for Azure Red Hat OpenShift",
			Version:     longVersion,
		},
		Host:     "management.azure.com",
		Schemes:  []string{"https"},
		Consumes: []string{"application/json"},
		Produces: []string{"application/json"},
		Paths:    populateTopLevelPaths("Microsoft.RedHatOpenShift", "openShiftCluster", "OpenShift cluster"),
		Definitions: Definitions{
			// TODO: this should be defined in the API package itself
			"OpenShiftClusterList": {
				Description: "OpenShiftClusterList represents a list of OpenShift clusters.",
				Properties: []NameSchema{
					{
						Name: "value",
						Schema: &Schema{
							Description: "The list of OpenShift clusters.",
							Type:        "array",
							Items: &Schema{
								Ref: "#/definitions/OpenShiftCluster",
							},
						},
					},
				},
			},
		},
		Parameters: ParametersDefinitions{
			"SubscriptionIdParameter": {
				Name:        "subscriptionId",
				In:          "path",
				Description: "Subscription credentials which uniquely identify Microsoft Azure subscription. The subscription ID forms part of the URI for every service call.",
				Required:    true,
				Type:        "string",
			},
			"ApiVersionParameter": {
				Name:        "api-version",
				In:          "query",
				Description: "Client API version.",
				Required:    true,
				Type:        "string",
			},
		},
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

	s.Paths["/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.RedHatOpenShift/openShiftClusters/{resourceName}/credentials"] = &PathItem{
		Post: &Operation{
			Tags:        []string{"OpenShiftClusters"},
			Summary:     "Gets credentials of a OpenShift cluster with the specified subscription, resource group and resource name.",
			Description: "Gets credentials of a OpenShift cluster with the specified subscription, resource group and resource name.  The operation returns the credentials.",
			OperationID: "OpenShiftClusters_GetCredentials",
			Parameters:  populateParameters(3, "OpenShiftCluster", "OpenShift cluster"),
			Responses:   populateResponses("OpenShiftClusterCredentials", false, http.StatusOK),
		},
	}

	s.Paths["/providers/Microsoft.RedHatOpenShift/operations"] = &PathItem{
		Get: &Operation{
			Tags:        []string{"Operations"},
			Summary:     "Lists all of the available RP operations.",
			Description: "Lists all of the available RP operations.  The operation returns the operations.",
			OperationID: "Operations_List",
			Parameters:  populateParameters(0, "Operation", "Operation"),
			Responses:   populateResponses("OperationList", false, http.StatusOK),
		},
	}

	err = define(s.Definitions, "github.com/jim-minter/rp/pkg/api/"+shortVersion, "OpenShiftCluster", "OpenShiftClusterCredentials")
	if err != nil {
		return err
	}

	err = define(s.Definitions, "github.com/jim-minter/rp/pkg/api", "CloudError", "OperationList")
	if err != nil {
		return err
	}

	s.Definitions["OpenShiftCluster"].AzureResource = true
	for i, property := range s.Definitions["OpenShiftCluster"].Properties {
		switch property.Name {
		case "name", "id", "type":
			property.Schema.ReadOnly = true
		case "location":
			property.Schema.Mutability = []string{"create", "read"}
		case "properties":
			property.Schema.ClientFlatten = true
		}
		s.Definitions["OpenShiftCluster"].Properties[i] = property
	}

	f := os.Stdout
	if outputFile != "" {
		var err error
		f, err = os.Create(outputFile)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	return e.Encode(s)
}
