package responsewriter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// ResponseWriter represents a ResponseWriter
type ResponseWriter interface {
	http.ResponseWriter
	Response() *http.Response
}

type responseWriter struct {
	bytes.Buffer
	r          *http.Request
	h          http.Header
	statusCode int
}

// New returns an http.ResponseWriter on which you can later call Response() to
// generate an *http.Response.
func New(r *http.Request) ResponseWriter {
	return &responseWriter{
		r:          r,
		h:          http.Header{},
		statusCode: http.StatusOK,
	}
}

func (w *responseWriter) Header() http.Header {
	return w.h
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *responseWriter) Response() *http.Response {
	return &http.Response{
		ProtoMajor: w.r.ProtoMajor,
		ProtoMinor: w.r.ProtoMinor,
		StatusCode: w.statusCode,
		Header:     w.h,
		Body:       ioutil.NopCloser(&w.Buffer),
	}
}
