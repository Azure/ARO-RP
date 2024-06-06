package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "fmt"

var _ fmt.Stringer = (*AccessToken)(nil)

type AccessToken struct {
	clusterID string
	token     string
}

func NewAccessToken(clusterID, token string) *AccessToken {
	return &AccessToken{
		clusterID: clusterID,
		token:     token,
	}
}

func (t *AccessToken) String() string {
	return fmt.Sprintf("%s:%s", t.clusterID, t.token)
}
