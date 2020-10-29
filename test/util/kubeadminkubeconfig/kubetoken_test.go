package kubeadminkubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestGetTokenURLFromConsoleURL(t *testing.T) {
	got, err := getTokenURLFromConsoleURL("https://console-openshift-console.apps.zhz4qvm8.as-rp-v4.osadev.cloud")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "https://oauth-openshift.apps.zhz4qvm8.as-rp-v4.osadev.cloud/oauth/authorize?client_id=openshift-challenging-client&response_type=token" {
		t.Error(got)
	}
}

func TestParseTokenResponse(t *testing.T) {
	location := "https://oauth-openshift.apps.zhz4qvm8.as-rp-v4.osadev.cloud/oauth/token/implicit#access_token=fIof3McZ5DVt1Uy6atsnUhis-y43dMctA5irrxH8ixk&expires_in=86400&scope=user%3Afull&token_type=Bearer"
	want := "fIof3McZ5DVt1Uy6atsnUhis-y43dMctA5irrxH8ixk"

	got, err := parseTokenResponse(location)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Error(got)
	}
}
