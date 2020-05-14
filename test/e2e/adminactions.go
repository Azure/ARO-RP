package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
)

func runAdminTestsInDevOnly() {
	if os.Getenv("RP_MODE") != "development" {
		Skip("RP_MODE not set to development, skipping admin actions tests")
	}
}

func adminRequest(method string, action string, body string, headers *http.Header, options ...string) ([]byte, error) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := os.Getenv("RESOURCEGROUP")
	clusterName := os.Getenv("CLUSTER")

	// This supports testing e2e in development and INT/prod environments.
	// Default to local development RP.
	adminURLPrefix := os.Getenv("ADMIN_URL_PREFIX")
	if adminURLPrefix == "" {
		adminURLPrefix = "https://localhost:8443/admin"
	}

	resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionID, resourceGroup, clusterName)
	adminURL, err := url.Parse(adminURLPrefix + resourceID + "/" + action)
	if err != nil {
		return nil, err
	}

	q := adminURL.Query()
	for _, opt := range options {
		optSplit := strings.Split(opt, "=")
		q.Set(optSplit[0], optSplit[1])
	}
	adminURL.RawQuery = q.Encode()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	fmt.Printf("adminRequest %s %s\n", method, adminURL.String())
	req, err := http.NewRequest(method, adminURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if headers != nil {
		*headers = resp.Header
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
