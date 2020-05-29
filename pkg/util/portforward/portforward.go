package portforward

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

// DialContext returns a connection to the specified cluster/namespace/pod/port.
func DialContext(ctx context.Context, log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, namespace, pod, port string) (net.Conn, error) {
	spdyConn, err := dialSpdy(ctx, env, oc, "/api/v1/namespaces/"+namespace+"/pods/"+pod+"/portforward")
	if err != nil {
		return nil, err
	}

	// Connect the error stream, r/o
	errorStream, err := spdyConn.CreateStream(http.Header{
		v1.StreamType:                 []string{v1.StreamTypeError},
		v1.PortHeader:                 []string{port},
		v1.PortForwardRequestIDHeader: []string{"0"},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}
	errorStream.Close() // this actually means CloseWrite()

	// Connect the data stream, r/w
	dataStream, err := spdyConn.CreateStream(http.Header{
		v1.StreamType:                 []string{v1.StreamTypeData},
		v1.PortHeader:                 []string{port},
		v1.PortForwardRequestIDHeader: []string{"0"},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}

	return newStreamConn(log, spdyConn, dataStream, errorStream), nil
}
