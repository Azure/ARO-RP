package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

func (mon *Monitor) requestMetricHTTP(ctx context.Context, pod string, endpoint string) (resp *http.Response, err error) {
	for i := 0; i < 3; i++ {
		hc := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					_, port, err := net.SplitHostPort(address)
					if err != nil {
						return nil, err
					}

					return portforward.DialContext(ctx, mon.log, mon.restconfig, "openshift-monitoring", fmt.Sprintf("%s-%d", pod, i), port)
				},
				// HACK: without this, keepalive connections don't get closed,
				// resulting in excessive open TCP connections, lots of
				// goroutines not exiting and memory not being freed.
				// TODO: consider persisting hc between calls to Monitor().  If
				// this is done, take care in the future to call
				// hc.CloseIdleConnections() when finally disposing of an hc.
				DisableKeepAlives: true,
			},
		}

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return
		}

		resp, err = hc.Do(req)
		if err == nil {
			break
		}
	}
	return
}
