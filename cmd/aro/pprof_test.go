package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestIsPprofEnabled(t *testing.T) {
	tests := []struct {
		name        string
		envPprof    string
		envRPMode   string
		wantEnabled bool
	}{
		{
			name:        "explicitly enabled",
			envPprof:    "true",
			envRPMode:   "",
			wantEnabled: true,
		},
		{
			name:        "explicitly enabled with 1",
			envPprof:    "1",
			envRPMode:   "",
			wantEnabled: true,
		},
		{
			name:        "explicitly disabled",
			envPprof:    "false",
			envRPMode:   "",
			wantEnabled: false,
		},
		{
			name:        "default in development mode",
			envPprof:    "",
			envRPMode:   "development",
			wantEnabled: true,
		},
		{
			name:        "default in production mode",
			envPprof:    "",
			envRPMode:   "",
			wantEnabled: false,
		},
		{
			name:        "case insensitive true",
			envPprof:    "TRUE",
			envRPMode:   "",
			wantEnabled: true,
		},
		{
			name:        "case insensitive development",
			envPprof:    "",
			envRPMode:   "DEVELOPMENT",
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origPprof := os.Getenv(envPprofEnabled)
			origRPMode := os.Getenv("RP_MODE")
			defer func() {
				os.Setenv(envPprofEnabled, origPprof)
				os.Setenv("RP_MODE", origRPMode)
			}()

			os.Setenv(envPprofEnabled, tt.envPprof)
			os.Setenv("RP_MODE", tt.envRPMode)

			got := isPprofEnabled()
			if got != tt.wantEnabled {
				t.Errorf("isPprofEnabled() = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}

func TestGetPprofPort(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantPort int
	}{
		{
			name:     "default port",
			envValue: "",
			wantPort: defaultPprofPort,
		},
		{
			name:     "custom port",
			envValue: "7070",
			wantPort: 7070,
		},
		{
			name:     "invalid port string",
			envValue: "invalid",
			wantPort: defaultPprofPort,
		},
		{
			name:     "port too low",
			envValue: "0",
			wantPort: defaultPprofPort,
		},
		{
			name:     "port too high",
			envValue: "65536",
			wantPort: defaultPprofPort,
		},
		{
			name:     "negative port",
			envValue: "-1",
			wantPort: defaultPprofPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origValue := os.Getenv(envPprofPort)
			defer os.Setenv(envPprofPort, origValue)

			os.Setenv(envPprofPort, tt.envValue)

			got := getPprofPort()
			if got != tt.wantPort {
				t.Errorf("getPprofPort() = %v, want %v", got, tt.wantPort)
			}
		})
	}
}

func TestGetPprofHost(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantHost string
	}{
		{
			name:     "default host",
			envValue: "",
			wantHost: defaultPprofHost,
		},
		{
			name:     "localhost",
			envValue: "localhost",
			wantHost: "localhost",
		},
		{
			name:     "127.0.0.1",
			envValue: "127.0.0.1",
			wantHost: "127.0.0.1",
		},
		{
			name:     "::1 ipv6",
			envValue: "::1",
			wantHost: "::1",
		},
		{
			name:     "non-localhost blocked",
			envValue: "0.0.0.0",
			wantHost: defaultPprofHost,
		},
		{
			name:     "external IP blocked",
			envValue: "192.168.1.1",
			wantHost: defaultPprofHost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origValue := os.Getenv(envPprofHost)
			defer os.Setenv(envPprofHost, origValue)

			os.Setenv(envPprofHost, tt.envValue)

			got := getPprofHost()
			if got != tt.wantHost {
				t.Errorf("getPprofHost() = %v, want %v", got, tt.wantHost)
			}
		})
	}
}

func TestIsLocalhostAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1", true},
		{"localhost", true},
		{"::1", true},
		{"[::1]", true},
		{"0.0.0.0", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := isLocalhostAddr(tt.addr)
			if got != tt.want {
				t.Errorf("isLocalhostAddr(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestNewPprofServerDisabled(t *testing.T) {
	// Save and restore environment
	origPprof := os.Getenv(envPprofEnabled)
	origRPMode := os.Getenv("RP_MODE")
	defer func() {
		os.Setenv(envPprofEnabled, origPprof)
		os.Setenv("RP_MODE", origRPMode)
	}()

	os.Setenv(envPprofEnabled, "false")
	os.Setenv("RP_MODE", "")

	log := logrus.NewEntry(logrus.New())
	srv, err := newPprofServer(log)

	if err != nil {
		t.Errorf("newPprofServer() error = %v, want nil", err)
	}
	if srv != nil {
		t.Errorf("newPprofServer() = %v, want nil when disabled", srv)
	}
}

func TestPprofServerStartStop(t *testing.T) {
	// Save and restore environment
	origPprof := os.Getenv(envPprofEnabled)
	origRPMode := os.Getenv("RP_MODE")
	origPort := os.Getenv(envPprofPort)
	defer func() {
		os.Setenv(envPprofEnabled, origPprof)
		os.Setenv("RP_MODE", origRPMode)
		os.Setenv(envPprofPort, origPort)
	}()

	os.Setenv(envPprofEnabled, "true")
	os.Setenv("RP_MODE", "")
	// Use a random high port to avoid conflicts
	os.Setenv(envPprofPort, "16060")

	log := logrus.NewEntry(logrus.New())
	srv, err := newPprofServer(log)
	if err != nil {
		t.Fatalf("newPprofServer() error = %v", err)
	}
	if srv == nil {
		t.Fatal("newPprofServer() returned nil")
	}

	ctx := context.Background()

	// Start the server
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify pprof endpoints are accessible
	resp, err := http.Get("http://127.0.0.1:16060/debug/pprof/")
	if err != nil {
		t.Errorf("Failed to access pprof index: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("pprof index returned status %d, want %d", resp.StatusCode, http.StatusOK)
		}
	}

	// Stop the server
	if err := srv.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Give the server time to stop
	time.Sleep(100 * time.Millisecond)

	// Verify the server is no longer responding
	_, err = http.Get("http://127.0.0.1:16060/debug/pprof/")
	if err == nil {
		t.Error("Server should not be responding after Stop()")
	}
}

func TestPprofServerPortCollision(t *testing.T) {
	// Save and restore environment
	origPprof := os.Getenv(envPprofEnabled)
	origRPMode := os.Getenv("RP_MODE")
	origPort := os.Getenv(envPprofPort)
	defer func() {
		os.Setenv(envPprofEnabled, origPprof)
		os.Setenv("RP_MODE", origRPMode)
		os.Setenv(envPprofPort, origPort)
	}()

	os.Setenv(envPprofEnabled, "true")
	os.Setenv("RP_MODE", "")
	os.Setenv(envPprofPort, "16061")

	log := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	// Start first server
	srv1, err := newPprofServer(log)
	if err != nil {
		t.Fatalf("newPprofServer() error = %v", err)
	}
	if err := srv1.Start(ctx); err != nil {
		t.Fatalf("First Start() error = %v", err)
	}
	defer srv1.Stop(ctx)

	// Try to start second server on same port
	srv2, err := newPprofServer(log)
	if err != nil {
		t.Fatalf("newPprofServer() error = %v", err)
	}

	err = srv2.Start(ctx)
	if err == nil {
		srv2.Stop(ctx)
		t.Error("Second Start() should have failed due to port collision")
	}
}

func TestPprofServerNilSafe(t *testing.T) {
	var srv *pprofServer
	ctx := context.Background()

	// These should not panic
	if err := srv.Start(ctx); err != nil {
		t.Errorf("Start() on nil server should return nil, got %v", err)
	}
	if err := srv.Stop(ctx); err != nil {
		t.Errorf("Stop() on nil server should return nil, got %v", err)
	}
}
