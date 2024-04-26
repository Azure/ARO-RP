package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleSecret() *Secret {
	doc := api.ExampleClusterManagerConfigurationDocumentSecret()
	ext := (&secretConverter{}).ToExternal(doc.Secret)
	return ext.(*Secret)
}

func ExampleSecretPutParameter() interface{} {
	s := exampleSecret()
	s.ID = ""
	s.Type = ""
	s.Name = ""
	return s
}

func ExampleSecretPatchParameter() interface{} {
	return ExampleSecretPutParameter()
}

func ExampleSecretResponse() interface{} {
	return exampleSecret()
}

func ExampleSecretListResponse() interface{} {
	return &SecretList{
		Secrets: []*Secret{
			ExampleSecretResponse().(*Secret),
		},
	}
}
