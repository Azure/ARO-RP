package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"reflect"

	v1 "k8s.io/api/core/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

type pullSecret struct {
	Auths map[string]map[string]interface{} `json:"auths,omitempty"`
}

type SerializedAuthMap struct {
	Auths map[string]SerializedAuth `json:"auths,omitempty"`
}

type SerializedAuth struct {
	Auth string `json:"auth"`
}

func UnmarshalSecretData(ps *v1.Secret) (*SerializedAuthMap, error) {
	var pullSecretData *SerializedAuthMap
	if data := ps.Data[v1.DockerConfigJsonKey]; len(data) > 0 {
		if err := json.Unmarshal(data, &pullSecretData); err != nil {
			return nil, err
		}
	}
	return pullSecretData, nil
}

func SetRegistryProfiles(_ps string, rps ...*api.RegistryProfile) (string, bool, error) {
	if _ps == "" {
		_ps = "{}"
	}

	var ps *pullSecret

	err := json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return "", false, err
	}

	if ps.Auths == nil {
		ps.Auths = map[string]map[string]interface{}{}
	}

	var changed bool

	for _, rp := range rps {
		v := map[string]interface{}{
			"auth": base64.StdEncoding.EncodeToString([]byte(rp.Username + ":" + string(rp.Password))),
		}

		if !reflect.DeepEqual(ps.Auths[rp.Name], v) {
			changed = true
		}
		ps.Auths[rp.Name] = v
	}

	b, err := json.Marshal(ps)
	return string(b), changed, err
}

// Merge returns _ps over _base.  If both _ps and _base have a given key, the
// version of it in _ps wins.
// TODO: Remove this in the next iteration, there are still parts of code using it
func Merge(_base, _ps string) (string, bool, error) {
	if _base == "" {
		_base = "{}"
	}

	if _ps == "" {
		_ps = "{}"
	}

	var base, ps *pullSecret

	err := json.Unmarshal([]byte(_base), &base)
	if err != nil {
		return "", false, err
	}

	err = json.Unmarshal([]byte(_ps), &ps)
	if err != nil {
		return "", false, err
	}

	var changed bool

	for k, v := range ps.Auths {
		if base.Auths == nil {
			base.Auths = map[string]map[string]interface{}{}
		}

		if !reflect.DeepEqual(base.Auths[k], v) {
			base.Auths[k] = v
			changed = true
		}
	}

	b, err := json.Marshal(base)
	return string(b), changed, err
}

// FixPullSecretData checks if the userData Secret Auth keys are not equal to the operator Auth keys
// if they are not they are considered broken and are replaced by the ones stored in operator Data
// if key is not found in operatorData it is not taken care of as this is user problem
// Function is idempotent
func FixPullSecretData(operatorData, userData *SerializedAuthMap) (fixedSecret *SerializedAuthMap, fixed bool) {
	fixedSecret = &SerializedAuthMap{
		Auths: map[string]SerializedAuth{},
	}

	for k, v := range userData.Auths {
		fixedSecret.Auths[k] = v
	}

	for k, v := range operatorData.Auths {
		userAuth, found := userData.Auths[k]

		if !found || userAuth.Auth != v.Auth {
			fixedSecret.Auths[k] = v
			fixed = true
			continue
		}
	}

	return fixedSecret, fixed
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

func Build(oc *api.OpenShiftCluster, ps string) (string, error) {
	pullSecret := os.Getenv("PULL_SECRET")

	pullSecret, _, err := Merge(pullSecret, ps)
	if err != nil {
		return "", err
	}

	pullSecret, _, err = SetRegistryProfiles(pullSecret, oc.Properties.RegistryProfiles...)
	if err != nil {
		return "", err
	}

	return pullSecret, nil
}
