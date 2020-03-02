package ready

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// SimpleHTTPClient to aid in mocking
type SimpleHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// URL returns boolean if url is ready
func URL(cli SimpleHTTPClient, url string) (bool, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := cli.Do(req)
	if err != nil {
		return false, err
	}
	return resp.StatusCode == http.StatusOK, nil
}

// URLPoolState returns when URL pool propagates to  expected state or an error after 20 min
func URLPoolState(ctx context.Context, log *logrus.Entry, cli SimpleHTTPClient, pool []string, expectedState bool) error {
	readyMap := make(map[string]bool, len(pool))
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, u := range pool {
			var err error
			readyMap[u], err = URL(cli, u)
			if err != nil {
				log.Info(err)
			}
			log.Infof("url %s state %t. Expected %t", u, readyMap[u], expectedState)
		}

		exit := true
		for _, r := range readyMap {
			if r != expectedState {

				exit = false
			}
		}
		return exit, nil
	}, ctx.Done())
}
