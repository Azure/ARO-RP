package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type API interface {
	GetClusterList(ctx context.Context, filter map[string]string) (*ClusterList, error)
	GetClusterUpgradePolicies(ctx context.Context, ocmClusterID string) (*UpgradePolicyList, error)
	CancelClusterUpgradePolicy(ctx context.Context, ocmClusterID, policyID string) (*CancelUpgradeResponse, error)
	GetClusterUpgradePolicyState(ctx context.Context, ocmClusterID, policyID string) (*UpgradePolicyState, error)
}

var _ API = (*Client)(nil)

type Client struct {
	httpClient *http.Client
	baseURL    string
	clusterID  string
}

func NewClient(clusterID, baseURL, token string) *Client {
	httpClient := &http.Client{
		Transport: NewAccessTokenTransport(NewAccessToken(clusterID, token)),
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		clusterID:  clusterID,
	}
}

func (c *Client) Send(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check the status code and handle non-2xx responses as errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unexpected response status code: %d, error reading response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("unexpected response status code: %d, body %s", resp.StatusCode, responseBody)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetClusterList(ctx context.Context, filter map[string]string) (*ClusterList, error) {
	if len(filter) == 0 {
		filter = map[string]string{
			"page":   "1",
			"size":   "1",
			"search": fmt.Sprintf("external_id='%s'", c.clusterID),
		}
	}

	rb := NewRequestBuilder(http.MethodGet, c.baseURL).
		SetContext(ctx).
		SetEndpoint(GetClusterListEndpointV1).
		AddHeader("Content-Type", "application/json").
		AddHeader("Accept", "application/json")

	for k, v := range filter {
		rb.AddParam(k, v)
	}

	request, err := rb.Build()
	if err != nil {
		return nil, err
	}

	clusterListBytes, err := c.Send(request)
	if err != nil {
		return nil, err
	}
	var clusterList *ClusterList
	if err := json.Unmarshal(clusterListBytes, &clusterList); err != nil {
		return nil, err
	}

	return clusterList, nil
}

func (c *Client) GetClusterUpgradePolicies(ctx context.Context, ocmClusterID string) (*UpgradePolicyList, error) {
	endpoint, err := BuildEndpoint(GetClusterUpgradePoliciesEndpointV1, map[string]string{
		"ocmClusterID": ocmClusterID,
	})
	if err != nil {
		return nil, err
	}

	rb := NewRequestBuilder(http.MethodGet, c.baseURL).
		SetContext(ctx).
		SetEndpoint(endpoint).
		AddHeader("Content-Type", "application/json").
		AddHeader("Accept", "application/json")

	request, err := rb.Build()
	if err != nil {
		return nil, err
	}

	upgradePolicyListBytes, err := c.Send(request)
	if err != nil {
		return nil, err
	}
	var upgradePolicyList *UpgradePolicyList
	if err := json.Unmarshal(upgradePolicyListBytes, &upgradePolicyList); err != nil {
		return nil, err
	}

	return upgradePolicyList, nil
}

func (c *Client) CancelClusterUpgradePolicy(ctx context.Context, ocmClusterID, policyID string) (*CancelUpgradeResponse, error) {
	cancelDescription := map[string]interface{}{
		"Value":       "cancelled",
		"Description": "Manually cancelled by SRE",
	}
	cancelDescriptionBody, _ := json.Marshal(cancelDescription)

	endpoint, err := BuildEndpoint(CancelClusterUpgradePolicyEndpointV1, map[string]string{
		"ocmClusterID": ocmClusterID,
		"policyID":     policyID,
	})
	if err != nil {
		return nil, err
	}

	rb := NewRequestBuilder(http.MethodPatch, c.baseURL).
		SetContext(ctx).
		SetEndpoint(endpoint).
		AddHeader("Content-Type", "application/json").
		AddHeader("Accept", "application/json").
		SetBody(cancelDescriptionBody)

	request, err := rb.Build()
	if err != nil {
		return nil, err
	}

	cancelUpgradeResponseBytes, err := c.Send(request)
	if err != nil {
		return nil, err
	}

	var cancelUpgradeResponse *CancelUpgradeResponse
	if err := json.Unmarshal(cancelUpgradeResponseBytes, &cancelUpgradeResponse); err != nil {
		return nil, err
	}

	return cancelUpgradeResponse, nil
}

func (c *Client) GetClusterUpgradePolicyState(ctx context.Context, ocmClusterID, policyID string) (*UpgradePolicyState, error) {
	endpoint, err := BuildEndpoint(GetClusterUpgradePolicyStateEndpointV1, map[string]string{
		"ocmClusterID": ocmClusterID,
		"policyID":     policyID,
	})
	if err != nil {
		return nil, err
	}

	rb := NewRequestBuilder(http.MethodGet, c.baseURL).
		SetContext(ctx).
		SetEndpoint(endpoint).
		AddHeader("Content-Type", "application/json").
		AddHeader("Accept", "application/json")

	request, err := rb.Build()
	if err != nil {
		return nil, err
	}

	upgradePolicyStateBytes, err := c.Send(request)
	if err != nil {
		return nil, err
	}
	var upgradePolicyState *UpgradePolicyState
	if err := json.Unmarshal(upgradePolicyStateBytes, &upgradePolicyState); err != nil {
		return nil, err
	}

	return upgradePolicyState, nil
}

func (c *Client) GetBaseURL() string {
	return c.baseURL
}
