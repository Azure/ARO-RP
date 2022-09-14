package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (g *generator) generateExamples(outputDir string, s *Swagger) error {
	err := os.RemoveAll(outputDir + "/examples")
	if err != nil {
		return err
	}

	err = os.MkdirAll(outputDir+"/examples", 0777)
	if err != nil {
		return err
	}

	for _, pi := range s.Paths {
		for _, op := range []*Operation{pi.Get, pi.Put, pi.Post, pi.Delete, pi.Options, pi.Head, pi.Patch} {
			if op == nil {
				continue
			}

			example := struct {
				Parameters NameParameters `json:"parameters,omitempty"`
				Responses  Responses      `json:"responses,omitempty"`
			}{
				Responses: Responses{},
			}
			for _, param := range op.Parameters {
				switch param := param.(type) {
				case Reference:
					switch param.Ref {
					case "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/ApiVersionParameter":
						example.Parameters = append(example.Parameters, NameParameter{
							Name:      "api-version",
							Parameter: stringutils.LastTokenByte(outputDir, '/'),
						})
					case "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/SubscriptionIdParameter":
						example.Parameters = append(example.Parameters, NameParameter{
							Name:      "subscriptionId",
							Parameter: "subscriptionId",
						})
					case "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/ResourceGroupNameParameter":
						example.Parameters = append(example.Parameters, NameParameter{
							Name:      "resourceGroupName",
							Parameter: "resourceGroup",
						})
					case "../../../../../common-types/resource-management/" + g.commonTypesVersion + "/types.json#/parameters/LocationParameter":
						example.Parameters = append(example.Parameters, NameParameter{
							Name:      "location",
							Parameter: "location",
						})
					}
				case Parameter:
					switch param.Name {
					case "resourceName":
						example.Parameters = append(example.Parameters, NameParameter{
							Name:      param.Name,
							Parameter: "resourceName",
						})
					case "childResourceName":
						example.Parameters = append(example.Parameters, NameParameter{
							Name:      "childResourceName",
							Parameter: "childResourceName",
						})
					case "parameters":
						switch param.Schema.Ref {
						case "#/definitions/OpenShiftCluster":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleOpenShiftClusterPutParameter(),
							})
						case "#/definitions/OpenShiftClusterUpdate":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleOpenShiftClusterPatchParameter(),
							})
						case "#/definitions/SyncSet":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleSyncSetPutParameter(),
							})
						case "#/definitions/SyncSetUpdate":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleSyncSetPatchParameter(),
							})
						case "#/definitions/MachinePool":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleMachinePoolPutParameter(),
							})
						case "#/definitions/MachinePoolUpdate":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleMachinePoolPatchParameter(),
							})
						case "#/definitions/SyncIdentityProvider":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleSyncIdentityProviderPutParameter(),
							})
						case "#/definitions/SyncIdentityProviderUpdate":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleSyncIdentityProviderPatchParameter(),
							})
						case "#/definitions/Secret":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleSecretPutParameter(),
							})
						case "#/definitions/SecretUpdate":
							example.Parameters = append(example.Parameters, NameParameter{
								Name:      param.Name,
								Parameter: g.exampleSecretPatchParameter(),
							})
						}
					}
				}
			}

			for statusCode, resp := range op.Responses {
				if statusCode == "default" {
					continue
				}

				response := resp.(Response)

				var body interface{}
				if response.Schema != nil {
					switch response.Schema.Ref {
					case "#/definitions/SyncSet":
						body = g.exampleSyncSetResponse()
					case "#/definitions/SyncSetList":
						body = g.exampleSyncSetListResponse()
					case "#/definitions/MachinePool":
						body = g.exampleSyncSetResponse()
					case "#/definitions/MachinePoolList":
						body = g.exampleSyncSetListResponse()
					case "#/definitions/SyncIdentityProvider":
						body = g.exampleSyncSetResponse()
					case "#/definitions/SyncIdentityProviderList":
						body = g.exampleSyncSetListResponse()
					case "#/definitions/Secret":
						body = g.exampleSecretResponse()
					case "#/definitions/SecretList":
						body = g.exampleSecretListResponse()
					case "#/definitions/OpenShiftCluster":
						body = g.exampleOpenShiftClusterResponse()
					case "#/definitions/OpenShiftClusterCredentials":
						body = g.exampleOpenShiftClusterCredentialsResponse()
					case "#/definitions/OpenShiftClusterAdminKubeconfig":
						body = g.exampleOpenShiftClusterAdminKubeconfigResponse()
					case "#/definitions/OpenShiftClusterList":
						body = g.exampleOpenShiftClusterListResponse()
					case "#/definitions/OperationList":
						body = g.exampleOperationListResponse()
					case "#/definitions/InstallVersions":
						body = g.exampleInstallVersions()
					}
				}

				example.Responses[statusCode] = struct {
					Body interface{} `json:"body,omitempty"`
				}{
					Body: body,
				}
			}

			b, err := json.MarshalIndent(example, "", "  ")
			if err != nil {
				return err
			}

			b = append(b, '\n')

			err = ioutil.WriteFile(outputDir+"/examples/"+op.OperationID+".json", b, 0666)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
