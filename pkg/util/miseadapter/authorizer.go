package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type MISEAdapter interface {
	IsAuthorized(ctx context.Context, r *http.Request) (bool, error)
	IsReady() bool
}

type miseAdapter struct {
	client *Client
	log    *logrus.Entry
}

func NewAuthorizer(miseAddress string, log *logrus.Entry) MISEAdapter {
	return &miseAdapter{
		client: New(http.DefaultClient, miseAddress),
		log:    log,
	}
}

func (m *miseAdapter) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	remoteAddr, _, _ := strings.Cut(r.RemoteAddr, ":")

	i := Input{
		OriginalUri:         fmt.Sprintf("http://%s%s", r.Host, r.URL.Path),
		OriginalMethod:      r.Method,
		OriginalIPAddress:   remoteAddr,
		AuthorizationHeader: r.Header.Get("Authorization"),
	}

	result, err := m.client.ValidateRequest(ctx, i, m.log)
	if err != nil {
		return false, err
	}

	if result.StatusCode != http.StatusOK {
		m.log.Errorf("mise authentication failed with %d: %s", result.StatusCode, result.ErrorDescription)
		return false, nil
	}

	return true, nil
}

func (m *miseAdapter) IsReady() bool {
	req, err := http.NewRequest(http.MethodGet, m.client.miseAddress+"/readyz", nil)
	if err != nil {
		m.log.Errorf("mise request create failed, %s", err)
		return false
	}
	m.client.httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, respErr := m.client.httpClient.Do(req)
	if respErr != nil {
		m.log.Errorf("mise readyz failed, %s", respErr)
		return false
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true
	}

	m.log.Errorf("mise readyz failed with %s status code", resp.StatusCode)
	return false
}
