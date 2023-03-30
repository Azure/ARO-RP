package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/env"
)

func adminRequest(ctx context.Context, method, path string, params url.Values, strict bool, in, out interface{}) (*http.Response, error) {
	if !env.IsLocalDevelopmentMode() {
		return nil, errors.New("only development RP mode is supported")
	}

	if params == nil {
		params = url.Values{}
	}

	params.Add("api-version", admin.APIVersion)

	adminAPIBaseURI := "https://localhost:8443"
	adminURL, err := url.Parse(adminAPIBaseURI + path)
	if err != nil {
		return nil, err
	}
	adminURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, method, adminURL.String(), nil)
	if err != nil {
		return nil, err
	}

	if in != nil {
		buf := &bytes.Buffer{}
		err := json.NewEncoder(buf).Encode(in)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(buf)
		req.Header.Set("Content-Type", "application/json")
	}

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
	defer func() {
		_, _ = resp.Body.Read(nil)
		_ = resp.Body.Close()
	}()

	if out != nil && resp.Header.Get("Content-Type") == "application/json" {
		decoder := json.NewDecoder(resp.Body)
		// If strict is set, enable DisallowUnknownFields. This is used to
		// verify that the response doesn't contain any fields that are not
		// defined, namely systemData.
		if strict {
			decoder.DisallowUnknownFields()
		}
		return resp, decoder.Decode(&out)
	} else if out != nil && resp.Header.Get("Content-Type") == "text/plain" {
		strOut := out.(*string)
		p, err := io.ReadAll(resp.Body)
		if err == nil {
			*strOut = string(p)
		}

		return resp, err
	}

	return resp, nil
}

// adminGetCluster returns admin representation of an ARO cluster
func adminGetCluster(g Gomega, ctx context.Context, resourceID string) *admin.OpenShiftCluster {
	var oc admin.OpenShiftCluster
	resp, err := adminRequest(ctx, http.MethodGet, resourceID, nil, true, nil, &oc)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
	return &oc
}

// adminListClusters returns a list of ARO clusters in admin representation.
// It handles pagination: function returns all the clusters from all pages.
func adminListClusters(g Gomega, ctx context.Context, path string) []*admin.OpenShiftCluster {
	ocs := make([]*admin.OpenShiftCluster, 0)
	params := url.Values{}
	for {
		var list admin.OpenShiftClusterList
		resp, err := adminRequest(ctx, http.MethodGet, path, params, true, nil, &list)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

		ocs = append(ocs, list.OpenShiftClusters...)

		if list.NextLink == "" {
			break
		}

		params = nextParams(g, list.NextLink)
	}
	return ocs
}

func nextParams(g Gomega, nextLink string) url.Values {
	url, err := url.Parse(nextLink)
	g.Expect(err).NotTo(HaveOccurred())

	return url.Query()
}
