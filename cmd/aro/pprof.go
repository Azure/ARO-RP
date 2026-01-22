package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	// Importing net/http/pprof registers the following handlers on http.DefaultServeMux:
	//   /debug/pprof/         - index page listing available profiles
	//   /debug/pprof/cmdline  - command line invocation
	//   /debug/pprof/profile  - CPU profile (accepts ?seconds=N)
	//   /debug/pprof/symbol   - symbol lookup
	//   /debug/pprof/trace    - execution trace (accepts ?seconds=N)
	//   /debug/pprof/heap     - heap profile
	//   /debug/pprof/goroutine - goroutine profile
	//   /debug/pprof/allocs   - allocation profile
	//   /debug/pprof/block    - block profile
	//   /debug/pprof/mutex    - mutex profile
	//   /debug/pprof/threadcreate - thread creation profile
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
)

const (
	defaultPprofPort  = 6060
	defaultPprofHost  = "127.0.0.1"
	pprofReadTimeout  = 30 * time.Second
	pprofWriteTimeout = 60 * time.Second
)

// pprofServer provides a production-ready pprof HTTP server with:
// - Environment variable configuration (PPROF_ENABLED, PPROF_PORT, PPROF_HOST)
// - Localhost-only binding and request validation for security
// - Port collision detection and graceful shutdown
type pprofServer struct {
	log      *logrus.Entry
	server   *http.Server
	listener net.Listener
	port     int
	host     string
}

func newPprofServer(log *logrus.Entry) (*pprofServer, error) {
	if !isPprofEnabled() {
		log.Info("pprof server disabled via environment variable")
		return nil, nil
	}

	return &pprofServer{
		log:  log,
		port: getPprofPort(),
		host: getPprofHost(),
	}, nil
}

func isPprofEnabled() bool {
	val := os.Getenv(envPprofEnabled)
	if val == "" {
		return strings.EqualFold(os.Getenv("RP_MODE"), "development")
	}
	return strings.EqualFold(val, "true") || val == "1"
}

func getPprofPort() int {
	if port, err := strconv.Atoi(os.Getenv(envPprofPort)); err == nil && port > 0 && port <= 65535 {
		return port
	}
	return defaultPprofPort
}

func getPprofHost() string {
	host := os.Getenv(envPprofHost)
	if host == "" || !isLocalhostAddr(host) {
		return defaultPprofHost
	}
	return host
}

func isLocalhostAddr(addr string) bool {
	return addr == "127.0.0.1" || addr == "localhost" || addr == "::1" || addr == "[::1]"
}

func (p *pprofServer) Start(ctx context.Context) error {
	if p == nil {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("pprof: failed to listen on %s: %w", addr, err)
	}
	p.listener = ln

	p.server = &http.Server{
		Handler:      p.localhostOnly(http.DefaultServeMux),
		ReadTimeout:  pprofReadTimeout,
		WriteTimeout: pprofWriteTimeout,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}

	p.log.Infof("pprof server listening on %s", addr)

	go func() {
		if err := p.server.Serve(p.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.log.Warnf("pprof server error: %v", err)
		}
	}()

	return nil
}

func (p *pprofServer) Stop(ctx context.Context) error {
	if p == nil || p.server == nil {
		return nil
	}
	p.log.Info("stopping pprof server")
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return p.server.Shutdown(shutdownCtx)
}

// localhostOnly rejects requests from non-localhost addresses
func (p *pprofServer) localhostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || !isLocalhostAddr(host) {
			p.log.Warnf("pprof: rejected request from %s", r.RemoteAddr)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
