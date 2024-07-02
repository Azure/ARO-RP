package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strconv"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/utils/strings/slices"
)

// populateParameters populates a parameters block.  Always expect an
// subscriptionId and apiVersion; the rest is dependent on specificity:
// n==0 list across provider
// n==1 list across subscription
// n==2 list across resource group
// n==3 action on resource not expecting input payload
// n==4 action on resource expecting input payload
// n==5 patch action on resource expecting input payload
// n==6 list across subscription and location
// n==7 action on child resource not expecting input payload
// n==8 action on child resource expecting input payload
// n==9 patch action on resource expecting input payload
// n==10 list child resources belonging to a parent resource

func (g *generator) populateParameters(n int, typ, friendlyName string) (s []interface{}) {
	s = []interface{}{
		Reference{
			Ref: "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/ApiVersionParameter",
		},
	}
	if n > 0 {
		s = append(s, Reference{
			Ref: "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/SubscriptionIdParameter",
		})
	}

	if n == 6 {
		s = append(s, Reference{
			Ref: "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/LocationParameter",
		})
		return
	}

	if n > 1 {
		s = append(s, Reference{
			Ref: "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/ResourceGroupNameParameter",
		})
	}

	if n > 2 {
		resourceNameParameter := Parameter{
			Name:        "resourceName",
			In:          "path",
			Description: "The name of the " + friendlyName + " resource.",
			Required:    true,
			Type:        "string",
		}
		if slices.Contains(proxyResources, friendlyName) {
			resourceNameParameter.Description = "The name of the OpenShift cluster resource."
			resourceNameParameter.Pattern = resourceNamePattern
			resourceNameParameter.MinLength = 1
			resourceNameParameter.MaxLength = 63
		}
		s = append(s, resourceNameParameter)
	}

	// gross. this is really hacky :/
	// this covers get,put,patch,delete by adding this
	// parameter as a required parameter for those operations
	// except when n==10, then its not a required parameter
	if n >= 7 && n != 10 {
		s = append(s, Parameter{
			Name:        "childResourceName",
			In:          "path",
			Description: "The name of the " + friendlyName + " resource.",
			Required:    true,
			Type:        "string",
			Pattern:     resourceNamePattern,
			MinLength:   1,
			MaxLength:   63,
		})
	}

	// TODO: refactor this entire function to make sense
	// so we can stop thinking about what int value builds a proper swagger parameter
	if n > 3 && n != 7 && n != 10 {
		s = append(s, Parameter{
			Name:        "parameters",
			In:          "body",
			Description: "The " + friendlyName + " resource.",
			Required:    true,
			Schema: &Schema{
				Ref: "#/definitions/" + typ,
			},
		})
	}

	if n == 5 || n == 9 {
		s[len(s)-1].(Parameter).Schema.Ref += "Update"
	}
	return
}

// populateResponses populates a responses block.  Always include the default
// error response.
func (g *generator) populateResponses(typ string, isDelete bool, statusCodes ...int) (responses map[string]interface{}) {
	responses = map[string]interface{}{
		"default": Response{
			Description: "Error response describing why the operation failed.  If the resource doesn't exist, 404 (Not Found) is returned.  If any of the input parameters is wrong, 400 (Bad Request) is returned.",
			Schema: &Schema{
				Ref: "#/definitions/CloudError",
			},
		},
	}

	for _, statusCode := range statusCodes {
		r := Response{
			Description: http.StatusText(statusCode),
		}
		if !isDelete {
			switch statusCode {
			case http.StatusOK, http.StatusCreated:
				r.Schema = &Schema{
					Ref: "#/definitions/" + typ,
				}
			}
		}
		responses[strconv.FormatInt(int64(statusCode), 10)] = r
	}

	return
}

// populateTopLevelPaths populates the paths for a top level ARM resource
func (g *generator) populateTopLevelPaths(resourceProviderNamespace, resourceType, friendlyName string) (ps Paths) {
	titleCaser := cases.Title(language.Und, cases.NoLower)
	ps = Paths{}

	ps["/subscriptions/{subscriptionId}/providers/"+resourceProviderNamespace+"/"+resourceType+"s"] = &PathItem{
		Get: &Operation{
			Tags:        []string{titleCaser.String(resourceType) + "s"},
			Summary:     "Lists " + friendlyName + "s in the specified subscription.",
			Description: "The operation returns properties of each " + friendlyName + ".",
			OperationID: titleCaser.String(resourceType) + "s_List",
			Parameters:  g.populateParameters(1, titleCaser.String(resourceType), friendlyName),
			Responses:   g.populateResponses(titleCaser.String(resourceType)+"List", false, http.StatusOK),
			Pageable: &Pageable{
				NextLinkName: "nextLink",
			},
		},
	}

	ps["/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/"+resourceProviderNamespace+"/"+resourceType+"s"] = &PathItem{
		Get: &Operation{
			Tags:        []string{titleCaser.String(resourceType) + "s"},
			Summary:     "Lists " + friendlyName + "s in the specified subscription and resource group.",
			Description: "The operation returns properties of each " + friendlyName + ".",
			OperationID: titleCaser.String(resourceType) + "s_ListByResourceGroup",
			Parameters:  g.populateParameters(2, titleCaser.String(resourceType), friendlyName),
			Responses:   g.populateResponses(titleCaser.String(resourceType)+"List", false, http.StatusOK),
			Pageable: &Pageable{
				NextLinkName: "nextLink",
			},
		},
	}

	ps["/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/"+resourceProviderNamespace+"/"+resourceType+"s/{resourceName}"] = &PathItem{
		Get: &Operation{
			Tags:        []string{titleCaser.String(resourceType) + "s"},
			Summary:     "Gets a " + friendlyName + " with the specified subscription, resource group and resource name.",
			Description: "The operation returns properties of a " + friendlyName + ".",
			OperationID: titleCaser.String(resourceType) + "s_Get",
			Parameters:  g.populateParameters(3, titleCaser.String(resourceType), friendlyName),
			Responses:   g.populateResponses(titleCaser.String(resourceType), false, http.StatusOK),
		},
		Put: &Operation{
			Tags:                 []string{titleCaser.String(resourceType) + "s"},
			Summary:              "Creates or updates a " + friendlyName + " with the specified subscription, resource group and resource name.",
			Description:          "The operation returns properties of a " + friendlyName + ".",
			OperationID:          titleCaser.String(resourceType) + "s_CreateOrUpdate",
			Parameters:           g.populateParameters(4, titleCaser.String(resourceType), friendlyName),
			Responses:            g.populateResponses(titleCaser.String(resourceType), false, http.StatusOK, http.StatusCreated),
			LongRunningOperation: true,
		},
		Delete: &Operation{
			Tags:                 []string{titleCaser.String(resourceType) + "s"},
			Summary:              "Deletes a " + friendlyName + " with the specified subscription, resource group and resource name.",
			Description:          "The operation returns nothing.",
			OperationID:          titleCaser.String(resourceType) + "s_Delete",
			Parameters:           g.populateParameters(3, titleCaser.String(resourceType), friendlyName),
			Responses:            g.populateResponses(titleCaser.String(resourceType), true, http.StatusAccepted, http.StatusNoContent),
			LongRunningOperation: true,
		},
		Patch: &Operation{
			Tags:                 []string{titleCaser.String(resourceType) + "s"},
			Summary:              "Creates or updates a " + friendlyName + " with the specified subscription, resource group and resource name.",
			Description:          "The operation returns properties of a " + friendlyName + ".",
			OperationID:          titleCaser.String(resourceType) + "s_Update",
			Parameters:           g.populateParameters(5, titleCaser.String(resourceType), friendlyName),
			Responses:            g.populateResponses(titleCaser.String(resourceType), false, http.StatusOK, http.StatusCreated),
			LongRunningOperation: true,
		},
	}

	return
}

func populateExamples(ps Paths) {
	for p, pi := range ps {
		for _, op := range []*Operation{pi.Get, pi.Put, pi.Post, pi.Delete, pi.Options, pi.Head, pi.Patch} {
			if op == nil {
				continue
			}
			op.Examples = map[string]Reference{
				op.Summary: {
					Ref: "./examples/" + op.OperationID + ".json",
				},
			}
		}

		ps[p] = pi
	}
}
