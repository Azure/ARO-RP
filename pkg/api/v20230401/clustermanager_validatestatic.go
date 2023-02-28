package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type clusterManagerStaticValidator struct{}

func (c clusterManagerStaticValidator) Static(body string, vars map[string]string) error {
	ocmResourceType := vars["ocmResourceType"]
	var resource map[string]interface{}

	if decodedBody, err := base64.StdEncoding.DecodeString(body); err == nil {
		err = json.Unmarshal(decodedBody, &resource)
		if err != nil {
			return err
		}
	} else {
		b := []byte(body)
		err := json.Unmarshal(b, &resource)
		if err != nil {
			return err
		}
	}

	payloadResourceKind := strings.ToLower(resource["kind"].(string))
	if payloadResourceKind != ocmResourceType {
		return fmt.Errorf("wanted Kind '%v', resource is Kind '%v'", ocmResourceType, payloadResourceKind)
	}

	return nil
}
