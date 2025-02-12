package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type simpleHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type internetChecker interface {
	Check(URLs []string) error
}

// checker evaluates our capability to create new
// connections to given internet endpoints.
type checker struct {
	checkTimeout time.Duration
	httpClient   simpleHTTPClient
}

func newInternetChecker() *checker {
	return &checker{
		checkTimeout: time.Minute,
		httpClient: &http.Client{
			Transport: &http.Transport{
				// We set DisableKeepAlives for two reasons:
				//
				// 1. If we're talking HTTP/2 and the remote end blackholes traffic,
				// Go has a bug whereby it doesn't reset the connection after a
				// timeout (https://github.com/golang/go/issues/36026).  If this
				// happens, we never have a chance to get healthy.  We have
				// specifically seen this with gcs.prod.monitoring.core.windows.net
				// in Korea Central, which currently has a bad server which when we
				// hit it causes our cluster creations to fail.
				//
				// 2. We *want* to evaluate our capability to successfully create
				// *new* connections to internet endpoints anyway.
				DisableKeepAlives: true,
			},
		},
	}
}
func (r *checker) Check(URLs []string) error {
	ch := make(chan error)
	checkCount := 0
	for _, url := range URLs {
		checkCount++
		go func(urlToCheck string) {
			ch <- r.checkWithRetry(urlToCheck)
		}(url)
	}

	errsAll := []string{}
	for i := 0; i < checkCount; i++ {
		if err := <-ch; err != nil {
			errsAll = append(errsAll, err.Error())
		}
	}
	if len(errsAll) != 0 {
		// TODO: Consider replacing with multi error wrapping with Go 1.20: https://github.com/golang/go/issues/53435#issuecomment-1320343377
		return fmt.Errorf("%s", strings.Join(errsAll, "\n"))
	}

	return nil
}

// checkWithRetry checks the URL, retrying a failed query a few times
func (r *checker) checkWithRetry(url string) error {
	var err error

	for i := 0; i < 6; i++ {
		err = r.checkOnce(url, r.checkTimeout/6)
		if err == nil {
			return nil
		}
	}

	return err
}

// checkOnce checks a given url.  The check both times out after a given timeout
// *and* will wait for the timeout if it fails, so that we don't hit endpoints
// too much.
func (r *checker) checkOnce(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		<-ctx.Done()
		return err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		<-ctx.Done()
		return fmt.Errorf("%s: %s", url, err)
	}

	resp.Body.Close()
	return nil
}
