package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
)

const testACR = "arosvc.azurecr.io"

func TestUserPasswordFromB64(t *testing.T) {
	for _, tt := range []struct {
		name               string
		input              string
		wantUserOutput     string
		wantPasswordOutput string
		wantErr            error
	}{
		{
			name:               "ok",
			input:              base64.StdEncoding.EncodeToString([]byte("user:password")),
			wantUserOutput:     "user",
			wantPasswordOutput: "password",
		},
		{
			name:    "no password",
			input:   base64.StdEncoding.EncodeToString([]byte("user:")),
			wantErr: fmt.Errorf("password string is not valid"),
		},
		{
			name:    "no password 2",
			input:   base64.StdEncoding.EncodeToString([]byte("nocolon")),
			wantErr: fmt.Errorf("password string is not valid"),
		},
		{
			name:    "no user",
			input:   base64.StdEncoding.EncodeToString([]byte(":password")),
			wantErr: fmt.Errorf("password string is not valid"),
		},
		{
			name:    "not valid b64",
			input:   "potato()+)(*",
			wantErr: fmt.Errorf("illegal base64 data at input byte 6"),
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			user, password, err := userPasswordFromB64(tt.input)
			if user != tt.wantUserOutput {
				t.Error(tt.name)
			}
			if password != tt.wantPasswordOutput {
				t.Error(tt.name)
			}
			if err != nil && tt.wantErr == nil {
				t.Error(tt.name)
			} else if err != nil && err.Error() != tt.wantErr.Error() {
				t.Error(tt.name)
			}
		})
	}
}

func TestBasicValidation(t *testing.T) {
	for _, tt := range []struct {
		name      string
		old       authsStruct
		new       authsStruct
		operation admissionv1.Operation
		required  map[string]bool
		wantIsOCM bool
		wantErr   error
	}{
		{
			name:      "creation of OCM secret",
			old:       authsStruct{},
			new:       authsStruct{Auths: map[string]Auth{ocmKey: {"credentials"}, "someregistry": {"credentials"}}},
			operation: admissionv1.Create,
			required:  map[string]bool{"someregistry": true},
			wantIsOCM: true,
			wantErr:   nil,
		},
		{
			name:      "creation of non OCM secret",
			old:       authsStruct{},
			new:       authsStruct{Auths: map[string]Auth{"someregistry": {"credentials"}}},
			operation: admissionv1.Create,
			required:  map[string]bool{"someregistry": true},
			wantIsOCM: false,
			wantErr:   nil,
		},
		{
			name:      "deletion of secret",
			old:       authsStruct{},
			new:       authsStruct{Auths: map[string]Auth{"someregistry": {"credentials"}}},
			operation: admissionv1.Delete,
			required:  map[string]bool{"someregistry": true},
			wantIsOCM: false,
			wantErr:   errors.New("cannot delete the ocm pullsecret"),
		},
		{
			name:      "removal of aro credential",
			old:       authsStruct{Auths: map[string]Auth{testACR: {"credentials"}}},
			new:       authsStruct{Auths: map[string]Auth{"someregistry": {"credentials"}}},
			operation: admissionv1.Update,
			required:  map[string]bool{"someregistry": true},
			wantIsOCM: false,
			wantErr:   errors.New("modification of arosvc.azurecr.io regisitry credentials is forbidden"),
		},
		{
			name:      "modification of aro credential",
			old:       authsStruct{Auths: map[string]Auth{testACR: {"credentials"}}},
			new:       authsStruct{Auths: map[string]Auth{testACR: {"potato"}}},
			operation: admissionv1.Update,
			required:  map[string]bool{"someregistry": true},
			wantIsOCM: false,
			wantErr:   errors.New("modification of arosvc.azurecr.io regisitry credentials is forbidden"),
		},
		{
			name:      "removal of ocm pullsecret",
			old:       authsStruct{Auths: map[string]Auth{ocmKey: {"credentials"}}},
			new:       authsStruct{Auths: map[string]Auth{"someregistry": {"credentials"}}},
			operation: admissionv1.Update,
			required:  map[string]bool{ocmKey: true},
			wantIsOCM: true,
			wantErr:   errors.New("the pullsecret does not have all the required registries"),
		},
		{
			name:      "missing registries",
			old:       authsStruct{Auths: map[string]Auth{}},
			new:       authsStruct{Auths: map[string]Auth{ocmKey: {"credentials"}}},
			operation: admissionv1.Update,
			required:  map[string]bool{"someregistry": true},
			wantIsOCM: true,
			wantErr:   errors.New("the pullsecret does not have all the required registries"),
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			isOCM, err := basicAuthValidation(tt.new, tt.old, tt.operation, tt.required, testACR)
			if isOCM != tt.wantIsOCM {
				t.Errorf("wanted %v but got %v", tt.wantIsOCM, isOCM)
			}
			fmt.Println(isOCM, err)
			if (tt.wantErr != nil && err.Error() != tt.wantErr.Error()) ||
				(tt.wantErr == nil && err != nil) {
				t.Errorf("wanted error to be %v, but got %v", tt.wantErr, err)
			}
		})
	}
}
