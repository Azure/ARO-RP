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

// A service that allows k8s port-forwarding, typically a pod.
type PortForwardService interface {
	GetPodName() string
	GetPodNamespace() string
	GetPodPort() string
}

// Supported port-forward sessions.
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

// Runs http requests on a k8s port-forward session and returns their responses.
func (p *portForwardActions) ForwardHttp(ctx context.Context, svc PortForwardService, httpReqs []*http.Request) ([]*http.Response, error) {
	hc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return portforward.DialContext(ctx, p.log, p.restconfig, svc.GetPodNamespace(), svc.GetPodName(), svc.GetPodPort())
			},
			// No need to reuse connections
			DisableKeepAlives: true,
		},
		Timeout: 3 * time.Minute,
	}

	var httpResps []*http.Response
	var resp *http.Response
	var err error

	for _, req := range httpReqs {
		resp, err = hc.Do(req)
		if err != nil {
			newErr := fmt.Errorf(
				"port forward to pod %s, in namespace %s and port %s failed: %v",
				svc.GetPodName(), svc.GetPodNamespace(), svc.GetPodPort(), err,
			)
			// note: callers must close the response body
			return httpResps, newErr
		}
		httpResps = append(httpResps, resp)
	}
	return httpResps, nil
}
