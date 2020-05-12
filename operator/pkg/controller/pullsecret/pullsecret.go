package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"path"
	"reflect"
)

type pullSecret struct {
	Auths map[string]map[string]interface{} `json:"auths,omitempty"`
}

// repair will parse the pull secret, for each registry, try read the registry secret from
// <secretPath>/<registry name> (like acrsvc.azurecr.io)
// overwrite the value and return if changed
func repair(_ps []byte, secretPath string) ([]byte, bool, error) {
	if _ps == nil || len(_ps) == 0 {
		_ps = []byte("{}")
	}

	var ps *pullSecret

	err := json.Unmarshal(_ps, &ps)
	if err != nil {
		return nil, false, err
	}

	if ps.Auths == nil {
		ps.Auths = map[string]map[string]interface{}{}
	}

	var changed bool

	files, err := ioutil.ReadDir(secretPath)
	if err != nil {
		return nil, false, err
	}

	for _, fName := range files {
		data, err := ioutil.ReadFile(path.Join(secretPath, fName.Name()))
		if err != nil {
			return nil, false, err
		}
		v := map[string]interface{}{
			"auth": base64.StdEncoding.EncodeToString(data),
		}

		if !reflect.DeepEqual(ps.Auths[fName.Name()], v) {
			changed = true
		}
		ps.Auths[fName.Name()] = v
	}

	b, err := json.Marshal(ps)
	return b, changed, err
}
