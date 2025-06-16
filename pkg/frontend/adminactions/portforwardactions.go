package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/sirupsen/logrus"
	restclient "k8s.io/client-go/rest"
)

type PortForwardService interface {
	GetPortForwardPodName() string
	GetPortForwardNamespace() string
	GetPortForwardPort() string
}

type PortForwardActions interface {
	ForwardHttp(ctx context.Context, portFwd PortForwardService, httpReqs []*http.Request) ([]*http.Response, error)
}

type portForwardActions struct {
	log        *logrus.Entry
	oc         *api.OpenShiftCluster
	restconfig *restclient.Config
}

func NewPortForwardActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster) (PortForwardActions, error) {
	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}

	return &portForwardActions{
		log:        log,
		oc:         oc,
		restconfig: restConfig,
	}, nil

}

// Establishes a k8s port forwarded connection and executes HTTP client requests on it.
func (p *portForwardActions) ForwardHttp(ctx context.Context, svc PortForwardService, httpReqs []*http.Request) ([]*http.Response, error) {
	hc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(address)
				if err != nil {
					p.log.Errorf("failed to split host and port from the address %s: %v", address, err)
					return nil, err
				}
				return portforward.DialContext(ctx, p.log, p.restconfig, svc.GetPortForwardNamespace(), svc.GetPortForwardPodName(), port)
			},
			DisableKeepAlives: true,
		},
		Timeout: time.Minute,
	}

	var httpResps []*http.Response
	var resp *http.Response
	var err error

	for _, req := range httpReqs {
		resp, err = hc.Do(req)
		if err != nil {
			newErr := fmt.Errorf(
				"port forward to pod %s, in namespace %s and port %s failed: %v",
				svc.GetPortForwardNamespace(), svc.GetPortForwardNamespace(), err,
			)
			return httpResps, newErr
		}
		httpResps = append(httpResps, resp)
	}
	return httpResps, nil
}
