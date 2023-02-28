package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"
)

var ocmResource = string(`
{
"apiVersion": "hive.openshift.io/v1",
"kind": "SyncSet",
"metadata": {
"name": "sample",
"namespace": "aro-f60ae8a2-bca1-4987-9056-f2f6a1837caa"
},
"spec": {
"clusterDeploymentRefs": [],
"resources": [
{
"apiVersion": "v1",
"kind": "ConfigMap",
"metadata": {
"name": "myconfigmap"
}
}
]
}
}
`)

var ocmResourceEncoded = "eyAKICAiYXBpVmVyc2lvbiI6ICJoaXZlLm9wZW5zaGlmdC5pby92MSIsCiAgImtpbmQiOiAiU3luY1NldCIsCiAgIm1ldGFkYXRhIjogewogICAgIm5hbWUiOiAic2FtcGxlIiwKICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LWYyZjZhMTgzN2NhYSIKICB9LAogICJzcGVjIjogewogICAgImNsdXN0ZXJEZXBsb3ltZW50UmVmcyI6IFtdLAogICAgInJlc291cmNlcyI6IFsKICAgICAgewogICAgICAgICJhcGlWZXJzaW9uIjogInYxIiwKICAgICAgICAia2luZCI6ICJDb25maWdNYXAiLAogICAgICAgICJtZXRhZGF0YSI6IHsKICAgICAgICAgICJuYW1lIjogIm15Y29uZmlnbWFwIgogICAgICAgIH0KICAgICAgfQogICAgXQogIH0KfQo="

func TestStatic(t *testing.T) {
	for _, tt := range []struct {
		name        string
		ocmResource string
		vars        map[string]string
		wantErr     bool
		err         string
	}{
		{
			name:        "payload Kind matches",
			ocmResource: ocmResource,
			vars: map[string]string{
				"ocmResourceType": "syncset",
			},
			wantErr: false,
		},
		{
			name:        "payload Kind matches and is a base64 encoded string",
			ocmResource: ocmResourceEncoded,
			vars: map[string]string{
				"ocmResourceType": "syncset",
			},
			wantErr: false,
		},
		{
			name:        "payload Kind does not match",
			ocmResource: ocmResource,
			vars: map[string]string{
				"ocmResourceType": "route",
			},
			wantErr: true,
			err:     "wanted Kind 'route', resource is Kind 'syncset'",
		},
		{
			name:        "payload Kind does not match and is a base64 encoded string",
			ocmResource: ocmResourceEncoded,
			vars: map[string]string{
				"ocmResourceType": "route",
			},
			wantErr: true,
			err:     "wanted Kind 'route', resource is Kind 'syncset'",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &clusterManagerStaticValidator{}

			err := c.Static(tt.ocmResource, tt.vars)
			if err != nil && tt.wantErr {
				if fmt.Sprint(err) != tt.err {
					t.Errorf("wanted '%v', got '%v'", tt.err, err)
				}
			}
		})
	}
}
