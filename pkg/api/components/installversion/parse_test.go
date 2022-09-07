package installversion

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
)

func TestParsePreInstallAPI(t *testing.T) {
	preInstallVersion := &v20200430.OpenShiftCluster{
		Properties: v20200430.OpenShiftClusterProperties{
			ClusterProfile: v20200430.ClusterProfile{
				Domain: "example",
			},
		},
	}

	b, err := json.Marshal(preInstallVersion)
	if err != nil {
		t.Fatal(err)
	}

	ver, err := FromExternalBytes(&b)
	if err != nil {
		t.Fatal(err)
	}

	if ver.Properties.InstallVersion != "" {
		t.Error(ver.Properties.InstallVersion)
	}
}

func TestParsePostInstallAPI(t *testing.T) {
	postInstallVersion := &v20220904.OpenShiftCluster{
		Properties: v20220904.OpenShiftClusterProperties{
			ClusterProfile: v20220904.ClusterProfile{
				Domain: "example",
			},
		},
	}

	b, err := json.Marshal(postInstallVersion)
	if err != nil {
		t.Fatal(err)
	}

	ver, err := FromExternalBytes(&b)
	if err != nil {
		t.Fatal(err)
	}

	if ver.Properties.InstallVersion != "" {
		t.Error(ver.Properties.InstallVersion)
	}

	postInstallVersionWithVersion := &v20220904.OpenShiftCluster{
		Properties: v20220904.OpenShiftClusterProperties{
			InstallVersion: "4.10.0",
			ClusterProfile: v20220904.ClusterProfile{
				Domain: "example",
			},
		},
	}

	b, err = json.Marshal(postInstallVersionWithVersion)
	if err != nil {
		t.Fatal(err)
	}

	ver, err = FromExternalBytes(&b)
	if err != nil {
		t.Fatal(err)
	}

	if ver.Properties.InstallVersion != "4.10.0" {
		t.Error(ver.Properties.InstallVersion)
	}
}
