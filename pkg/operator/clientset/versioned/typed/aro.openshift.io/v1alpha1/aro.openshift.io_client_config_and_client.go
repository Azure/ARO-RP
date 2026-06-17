package v1alpha1

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	rest "k8s.io/client-go/rest"
)

// NewForConfigAndClient creates a new AroV1alpha1Client for the given config and
// http client.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*AroV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &AroV1alpha1Client{client}, nil
}
