package encrypt

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/util/manifest"
	"github.com/ugorji/go/codec"
)

func TestUnmarshalSecure(t *testing.T) {
	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	cipher, err := New(make([]byte, 32))
	if err != nil {
		t.Error(err)
	}

	err = AddExtensions(&h.BasicHandle, cipher)
	if err != nil {
		t.Error(err)
	}

	for _, tt := range []struct {
		name   string
		modify func(doc *api.OpenShiftClusterDocument)
	}{
		{
			name: "noop",
		},
		{
			name: "rsa.PrivateKey",
			modify: func(doc *api.OpenShiftClusterDocument) {
				privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Error(err)
				}
				doc.OpenShiftCluster.Properties.SSHKey = privateKey
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := manifest.ValidOpenShiftClusterDocument()
			if tt.modify != nil {
				tt.modify(input)
			}

			buf := &bytes.Buffer{}
			err = codec.NewEncoder(buf, h).Encode(input)
			if err != nil {
				t.Error(err)
			}
			data, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Error(err)
			}

			output := &api.OpenShiftClusterDocument{}
			err = codec.NewDecoder(bytes.NewReader(data), h).Decode(output)
			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(output, input) {
				inputB, _ := json.Marshal(input)
				outputB, _ := json.Marshal(output)
				t.Errorf("wants: %s \n , got: %s \n ", string(inputB), string(outputB))
			}
		})
	}
}
