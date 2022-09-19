package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExampleInstallVersionResponse() interface{} {
	version := InstallVersion{
		proxyResource: true,
		Name:          "default",
		Properties: InstallVersionProperties{
			Version: "4.10.0",
		},
	}
	return version
}

// ExampleInstallVersion returns an example
// InstallVersion object i.e []string that the RP would return to an end-user.
func ExampleInstallVersionListResponse() interface{} {
	return &InstallVersionList{
		InstallVersions: []*InstallVersion{
			ExampleInstallVersionResponse().(*InstallVersion),
		},
	}
}
