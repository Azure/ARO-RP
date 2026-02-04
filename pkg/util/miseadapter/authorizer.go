package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	MISE_CONNECTION_TIMEOUT = time.Second * 5
	MISE_RETRY_DELAY        = time.Millisecond * 100
	MISE_RETRY_COUNT        = 3
)

type MISEAdapter interface {
	IsAuthorized(log *logrus.Entry, r *http.Request) (bool, error)
	IsReady() bool
}

type miseAdapter struct {
	client *Client
	log    *logrus.Entry
	sleep  func(time.Duration)
}

func NewAuthorizer(miseAddress string, log *logrus.Entry) *miseAdapter {
	return &miseAdapter{
		client: New(&http.Client{
			Transport: &http.Transport{
				// disable HTTP/2 for now due to timeout issues:
				// https://github.com/golang/go/issues/36026
				Protocols: nil,
			},
			Timeout: MISE_CONNECTION_TIMEOUT,
		}, miseAddress),
		log:   log.WithField("component", "miseadapter"),
		sleep: time.Sleep,
	}
}

func (m *miseAdapter) IsAuthorized(log *logrus.Entry, r *http.Request) (bool, error) {
	ctx := r.Context()

	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false, fmt.Errorf("invalid remote address %q: %w", r.RemoteAddr, err)
	}

	i := Input{
		OriginalUri:         fmt.Sprintf("http://%s%s", r.Host, r.URL.Path),
		OriginalMethod:      r.Method,
		OriginalIPAddress:   remoteAddr,
		AuthorizationHeader: r.Header.Get("Authorization"),
	}

	var result Result

	for attempt := range MISE_RETRY_COUNT {
		if attempt > 0 {
			// Linear backoff: 100ms, 200ms for retries
			sleepDuration := MISE_RETRY_DELAY * time.Duration(attempt)
			log.Infof("mise authentication retry attempt %d/%d after %v", attempt+1, MISE_RETRY_COUNT, sleepDuration)
			m.sleep(sleepDuration)
		}

		result, err = m.client.ValidateRequest(ctx, i, log)
		if err != nil {
			// fail without retry if te context is cancelled
			if errors.Is(err, context.Canceled) {
				return false, err
			}

			// Retry on network errors or context deadline exceeded
			log.Warnf("mise authentication attempt %d/%d failed with error: %v", attempt+1, MISE_RETRY_COUNT, err)
			if attempt < MISE_RETRY_COUNT-1 {
				continue
			}
			return false, err
		}

		if result.StatusCode == http.StatusOK {
			if attempt > 0 {
				log.Infof("mise authentication succeeded on attempt %d/%d", attempt+1, MISE_RETRY_COUNT)
			}
			return true, nil
		}

		// Don't retry on non-transient errors (4xx client errors except for specific cases)
		if result.StatusCode >= 400 && result.StatusCode < 500 && result.StatusCode != http.StatusRequestTimeout && result.StatusCode != http.StatusTooManyRequests {
			log.Errorf("mise authentication failed with %d: %s", result.StatusCode, result.ErrorDescription)
			return false, nil
		}

		// Retry on 5xx server errors, 408 Request Timeout, and 429 Too Many Requests
		log.Warnf("mise authentication attempt %d/%d failed with status %d: %s", attempt+1, MISE_RETRY_COUNT, result.StatusCode, result.ErrorDescription)
	}

	// All retries exhausted
	log.Errorf("mise authentication failed after %d attempts with %d: %s", MISE_RETRY_COUNT, result.StatusCode, result.ErrorDescription)
	return false, nil
}

func (m *miseAdapter) IsReady() bool {
	ready, _ := m.isReady()
	return ready
}

func (m *miseAdapter) isReady() (bool, error) {
	req, err := http.NewRequest(http.MethodGet, m.client.miseAddress+"/readyz", nil)
	if err != nil {
		m.log.Errorf("mise request create failed, %s", err)
		return false, err
	}
	resp, respErr := m.client.httpClient.Do(req)
	if respErr != nil {
		m.log.Errorf("mise readyz failed, %s", respErr)
		return false, respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	m.log.Errorf("mise readyz failed with %d status code", resp.StatusCode)
	return false, fmt.Errorf("status code %d", resp.StatusCode)
}
