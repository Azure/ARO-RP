package portforward

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

// DialContext returns a connection to the specified cluster/namespace/pod/port.
func DialContext(ctx context.Context, log *logrus.Entry, restconfig *rest.Config, namespace, pod, port string) (net.Conn, error) {
	spdyConn, err := dialSpdy(ctx, restconfig, "/api/v1/namespaces/"+namespace+"/pods/"+pod+"/portforward")
	if err != nil {
		return nil, err
	}

	// Connect the error stream, r/o
	errorStream, err := spdyConn.CreateStream(http.Header{
		corev1.StreamType:                 []string{corev1.StreamTypeError},
		corev1.PortHeader:                 []string{port},
		corev1.PortForwardRequestIDHeader: []string{"0"},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}
	errorStream.Close() // this actually means CloseWrite()

	// Connect the data stream, r/w
	dataStream, err := spdyConn.CreateStream(http.Header{
		corev1.StreamType:                 []string{corev1.StreamTypeData},
		corev1.PortHeader:                 []string{port},
		corev1.PortForwardRequestIDHeader: []string{"0"},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}

	return NewStreamConn(log, spdyConn, dataStream, errorStream), nil
}
