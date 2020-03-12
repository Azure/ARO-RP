package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"

	"github.com/Azure/ARO-RP/pkg/api"
)

type RegistryAuth struct {
	Auth string `json:"auth,omitempty"`
}

type PullSecret struct {
	Auths map[string]RegistryAuth `json:"auths,omitempty"`
}

func SetRegistryAuth(original string, rp *api.RegistryProfile) (string, error) {
	var pr *PullSecret
	err := json.Unmarshal([]byte(original), &pr)
	if err != nil {
		return "", err
	}
	pr.Auths[rp.Name] = RegistryAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(rp.Username + ":" + string(rp.Password))),
	}
	data, err := json.Marshal(pr)
	return string(data), err
}
