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

	const maxRetries = 3
	const retryDelayMs = 100

	var result Result
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Linear backoff: 100ms, 200ms for retries
			sleepDuration := time.Duration(attempt*retryDelayMs) * time.Millisecond
			m.log.Infof("mise authentication retry attempt %d/%d after %v", attempt+1, maxRetries, sleepDuration)
			time.Sleep(sleepDuration)
		}

		result, err = m.client.ValidateRequest(ctx, i, m.log)
		if err != nil {
			// Retry on network errors or context deadline exceeded
			m.log.Warnf("mise authentication attempt %d/%d failed with error: %v", attempt+1, maxRetries, err)
			if attempt < maxRetries-1 {
				continue
			}
			return false, err
		}

		if result.StatusCode == http.StatusOK {
			if attempt > 0 {
				m.log.Infof("mise authentication succeeded on attempt %d/%d", attempt+1, maxRetries)
			}
			return true, nil
		}

		// Don't retry on non-transient errors (4xx client errors except for specific cases)
		if result.StatusCode >= 400 && result.StatusCode < 500 && result.StatusCode != http.StatusRequestTimeout && result.StatusCode != http.StatusTooManyRequests {
			m.log.Errorf("mise authentication failed with %d: %s", result.StatusCode, result.ErrorDescription)
			return false, nil
		}

		// Retry on 5xx server errors, 408 Request Timeout, and 429 Too Many Requests
		m.log.Warnf("mise authentication attempt %d/%d failed with status %d: %s", attempt+1, maxRetries, result.StatusCode, result.ErrorDescription)
	}

	// All retries exhausted
	m.log.Errorf("mise authentication failed after %d attempts with %d: %s", maxRetries, result.StatusCode, result.ErrorDescription)
	return false, nil
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

	m.log.Errorf("mise readyz failed with %d status code", resp.StatusCode)
	return false
}
