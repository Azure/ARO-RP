package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Portal represents a portal
type Portal struct {
	MissingFields

	Username string `json:"username"`
	ID       string `json:"id,omitempty"`

	SSH        *SSH        `json:"ssh,omitempty"`
	Kubeconfig *Kubeconfig `json:"kubeconfig,omitempty"`
}

type SSH struct {
	MissingFields

	Master int `json:"master,omitempty"`
}

type Kubeconfig struct {
	MissingFields

	Elevated bool `json:"elevated,omitempty"`
}
