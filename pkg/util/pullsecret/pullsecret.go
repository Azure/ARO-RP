package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"reflect"

	"github.com/Azure/ARO-RP/pkg/api"
)

type pullSecret struct {
	Auths map[string]map[string]interface{} `json:"auths,omitempty"`
}

func Replace(_ps []byte, secrets map[string]string) ([]byte, bool, error) {
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
	for repo, secret := range secrets {
		v := map[string]interface{}{
			"auth": secret,
		}

		if !reflect.DeepEqual(ps.Auths[repo], v) {
			changed = true
		}
		ps.Auths[repo] = v
	}

	b, err := json.Marshal(ps)
	return b, changed, err
}

func Auths(_ps []byte) (map[string]map[string]interface{}, error) {
	if _ps == nil || len(_ps) == 0 {
		return nil, nil
	}

	var ps *pullSecret

	err := json.Unmarshal(_ps, &ps)
	if err != nil {
		return nil, err
	}

	return ps.Auths, nil
}

func SetRegistryProfiles(_ps string, rps ...*api.RegistryProfile) (string, bool, error) {
	secrets := map[string]string{}
	for _, rp := range rps {
		secrets[rp.Name] = base64.StdEncoding.EncodeToString([]byte(rp.Username + ":" + string(rp.Password)))
	}

	b, changed, err := Replace([]byte(_ps), secrets)
	return string(b), changed, err
}

// Merge returns _ps over _base.  If both _ps and _base have a given key, the
// version of it in _ps wins.
func Merge(_base, _ps string) (string, error) {
	if _base == "" {
		_base = "{}"
	}

	if _ps == "" {
		_ps = "{}"
	}

	var base, ps *pullSecret

	err := json.Unmarshal([]byte(_base), &base)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return "", err
	}

	for k, v := range ps.Auths {
		if base.Auths == nil {
			base.Auths = map[string]map[string]interface{}{}
		}

		base.Auths[k] = v
	}

	b, err := json.Marshal(base)
	return string(b), err
}

func RemoveKey(_ps, key string) (string, error) {
	if _ps == "" {
		_ps = "{}"
	}

	var ps *pullSecret

	err := json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return "", err
	}

	delete(ps.Auths, key)

	b, err := json.Marshal(ps)
	return string(b), err
}

func Validate(_ps string) error {
	if _ps == "" {
		_ps = "{}"
	}

	var ps *pullSecret

	return json.Unmarshal([]byte(_ps), &ps)
}
