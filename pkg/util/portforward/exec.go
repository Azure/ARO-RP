package portforward

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

// ExecStdout executes a command in the given namespace/pod/container and
// streams its stdout.
func ExecStdout(ctx context.Context, log *logrus.Entry, restconfig *rest.Config, namespace, pod, container string, command []string) (io.ReadCloser, error) {
	v := url.Values{
		"container": []string{container},
		"command":   command,
		"stdout":    []string{"true"},
	}

	spdyConn, err := dialSpdy(ctx, restconfig, "/api/v1/namespaces/"+namespace+"/pods/"+pod+"/exec?"+v.Encode())
	if err != nil {
		return nil, err
	}

	// Connect the error stream, r/o
	errorStream, err := spdyConn.CreateStream(http.Header{
		corev1.StreamType: []string{corev1.StreamTypeError},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}
	errorStream.Close() // this actually means CloseWrite()

	// Connect the stdout stream, r/o
	stdoutStream, err := spdyConn.CreateStream(http.Header{
		corev1.StreamType: []string{corev1.StreamTypeStdout},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}
	stdoutStream.Close() // this actually means CloseWrite()

	return NewStreamConn(log, spdyConn, stdoutStream, errorStream), nil
}
