package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/Azure/go-autorest/autorest"
)

// LoggingSender intercepts requests and logs them out.
// Usage: client.Sender = &LoggingSender{client.Sender}
type LoggingSender struct {
	autorest.Sender
}

func (ls *LoggingSender) Do(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Del("Authorization")
	b, _ := httputil.DumpRequestOut(clone, true)
	fmt.Fprintf(os.Stderr, "%s\n\n", string(b))
	resp, err := ls.Sender.Do(req)
	if resp != nil {
		b, _ = httputil.DumpResponse(resp, true)
		fmt.Fprintf(os.Stderr, "%s\n\n", string(b))
	}
	return resp, err
}
