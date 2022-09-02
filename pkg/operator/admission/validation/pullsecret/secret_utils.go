package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

type authsStruct struct {
	Auths map[string]Auth `json:"auths"`
}

type Auth struct {
	Auth string `json:"auth"`
}

//unmarshalRequestToSecret extracts secret and oldsecret (if they are defined) from the admission request
func unmarshalRequestToSecret(request *admissionv1.AdmissionRequest) (corev1.Secret, corev1.Secret, error) {
	secret := corev1.Secret{}
	oldSecret := corev1.Secret{}
	var err error
	//new object not defined on deletion
	if request.Operation != admissionv1.Delete {
		err = json.Unmarshal(request.Object.Raw, &secret)

		if err != nil {
			return corev1.Secret{}, corev1.Secret{}, err
		}
	}

	if request.Operation == admissionv1.Update || request.Operation == admissionv1.Delete {
		err = json.Unmarshal(request.OldObject.Raw, &oldSecret)
	}
	return secret, oldSecret, err
}

func modifiedAROSecret(new, old authsStruct, azureRegistry string) bool {
	//we only care for deletion and modifiction, creation is ignored

	//secret is deleted
	if new.Auths == nil && old.Auths != nil {
		return true
	}

	//secret is modified
	if (old.Auths != nil && new.Auths != nil) &&
		(new.Auths[azureRegistry].Auth != old.Auths[azureRegistry].Auth) {
		return true
	}
	return false
}

func secretIsOCM(new, old authsStruct) bool {
	//pullsecret is for OCM only if it has the "cloud.openshift.com" cred
	//if it is a create op, there is no old.
	//if it is a delete there is no new

	if new.Auths != nil {
		if _, ok := new.Auths[ocmKey]; ok {
			return true
		}
	}

	if old.Auths != nil {
		if _, ok := old.Auths[ocmKey]; ok {
			return true
		}
	}

	return false
}

func authsStructFromSecret(secret corev1.Secret) (authsStruct, error) {
	credentials, err := extractCredentialsFromSecret(secret, secret.Name)
	if err != nil {
		return authsStruct{}, err
	}

	return jsonToAuthStruct(credentials)
}

func extractCredentialsFromSecret(secret corev1.Secret, name string) ([]byte, error) {
	if v, ok := secret.Data[".dockerconfigjson"]; ok {
		return v, nil
	}
	if v, ok := secret.Data[".dockercfg"]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("%s did not have a dockerconfigjson or dockercfg field", name)
}

//userPasswordFromB64 extracts username and password from the
//base64 encoded field
func userPasswordFromB64(encoded string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}
	decodedString := string(decoded)
	index := strings.Index(decodedString, ":")
	if index <= 0 || index == len(decodedString)-1 {
		return "", "", fmt.Errorf("password string is not valid")
	}
	return decodedString[:index], decodedString[index+1:], nil
}

func jsonToAuthStruct(jsonBytes []byte) (authsStruct, error) {
	authStruct := authsStruct{}
	err := json.Unmarshal(jsonBytes, &authStruct)
	return authStruct, err
}

func basicAuthValidation(new, old authsStruct, operation admissionv1.Operation, required map[string]bool, azureRegistry string) (isOCM bool, err error) {
	if operation == admissionv1.Delete {
		return false, errors.New("cannot delete the ocm pullsecret")
	}

	if modifiedAROSecret(new, old, azureRegistry) {
		return false, fmt.Errorf("modification of %s regisitry credentials is forbidden", azureRegistry)
	}

	isOCM = secretIsOCM(new, old)
	if isOCM && !new.hasAllRequired(required) {
		return isOCM, errors.New("the pullsecret does not have all the required registries")
	}

	return isOCM, nil
}

//extractAuthsFromSecrets extracts auths and oldauth from secret and oldsecret
func extractAuthsFromSecrets(secret, oldSecret corev1.Secret, operation admissionv1.Operation) (authsStruct, authsStruct, error) {
	authsStructNew := authsStruct{}
	var err error
	if operation != admissionv1.Delete {
		authsStructNew, err = authsStructFromSecret(secret)

		if err != nil {
			return authsStruct{}, authsStruct{}, err
		}
	}

	authsStructOld := authsStruct{}
	if operation == admissionv1.Delete || operation == admissionv1.Update {
		authsStructOld, err = authsStructFromSecret(oldSecret)
		if err != nil {
			return authsStruct{}, authsStruct{}, err
		}
	}

	return authsStructNew, authsStructOld, nil
}

//hasAllRequired checks that the pullsecret has all the required registries
func (a *authsStruct) hasAllRequired(required map[string]bool) bool {
	requiredPresent := 0
	for k := range a.Auths {
		if required[k] {
			requiredPresent++
		}
	}

	return requiredPresent == len(required)
}
