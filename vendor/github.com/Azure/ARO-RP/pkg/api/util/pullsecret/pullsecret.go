package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
)

type pullSecret struct {
	Auths map[string]map[string]interface{} `json:"auths,omitempty"`
}

func Validate(_ps string) error {
	if _ps == "" {
		_ps = "{}"
	}

	var ps *pullSecret

	return json.Unmarshal([]byte(_ps), &ps)
}
