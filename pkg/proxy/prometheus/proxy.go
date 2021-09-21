package prometheus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"net/http"

	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/sirupsen/logrus"
)

type prometheusRT struct {
	log *logrus.Entry

	restconfig *rest.Config
}

type PrometheusRoundTripper interface {
	RoundTripper(r *http.Request) (*http.Response, error)
}

func (p *prometheusRT) cli() (*http.Client, error) {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return portforward.DialContext(ctx, p.log, p.restconfig, "openshift-monitoring", "prometheus-k8s-0", "9090")
			},
			// HACK: without this, keepalive connections don't get closed,
			// resulting in excessive open TCP connections, lots of
			// goroutines not exiting and memory not being freed.
			// TODO: consider persisting hc between calls to Monitor().  If
			// this is done, take care in the future to call
			// hc.CloseIdleConnections() when finally disposing of an hc.
			DisableKeepAlives: true,
		},
	}, nil
}

func (p *prometheusRT) RoundTripper(r *http.Request) (*http.Response, error) {
	cli, _ := p.cli()
	return cli.Do(r)
}
