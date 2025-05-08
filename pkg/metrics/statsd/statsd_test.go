package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"net"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEmitGauge(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Location().AnyTimes().Return("eastus")
	env.EXPECT().Hostname().AnyTimes().Return("test-host")

	c1, c2 := net.Pipe()

	s := &statsd{
		env: env,

		account:   "*",
		namespace: "*",

		conn: c1,
		ch:   make(chan *metric),

		now: func() time.Time { return time.Time{} },
	}

	go s.run()

	s.EmitGauge("tests.test_key", 42, map[string]string{"key": "value"})

	m, err := bufio.NewReader(c2).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if m != `{"Metric":"tests.test_key","Account":"*","Namespace":"*","Dims":{"hostname":"test-host","key":"value","location":"eastus"},"TS":"0001-01-01T00:00:00.000"}:42|g`+"\n" {
		t.Error(m)
	}
}

func TestEmitFloat(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Location().AnyTimes().Return("eastus")
	env.EXPECT().Hostname().AnyTimes().Return("test-host")

	c1, c2 := net.Pipe()

	s := &statsd{
		env: env,

		account:   "*",
		namespace: "*",

		conn: c1,
		ch:   make(chan *metric),

		now: func() time.Time { return time.Time{} },
	}

	go s.run()

	s.EmitFloat("tests.test_key", 5, map[string]string{"key": "value"})

	m, err := bufio.NewReader(c2).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if m != `{"Metric":"tests.test_key","Account":"*","Namespace":"*","Dims":{"hostname":"test-host","key":"value","location":"eastus"},"TS":"0001-01-01T00:00:00.000"}:5.000000|f`+"\n" {
		t.Error(m)
	}
}

func TestParseSocketEnv(t *testing.T) {
	for _, tt := range []struct {
		name        string
		teststring  string
		wantNetwork string
		wantAddress string
		wantError   string
	}{
		{
			name:        "Valid string",
			teststring:  "foo:bar",
			wantNetwork: "foo",
			wantAddress: "bar",
		},
		{
			name:        "Empty network part-regarded valid",
			teststring:  ":bar",
			wantNetwork: "",
			wantAddress: "bar",
		},
		{
			name:        "Empty address part-regarded valid",
			teststring:  "foo:",
			wantNetwork: "foo",
			wantAddress: "",
		},
		{
			name:       "No separator",
			teststring: "somerandomtext",
			wantError:  "malformed definition for the mdm statds socket. Expecting udp:<hostname>:<port> or unix:<path-to-socket> format. Got: \"somerandomtext\"",
		},
		{
			name:       "Empty string",
			teststring: "",
			wantError:  "malformed definition for the mdm statds socket. Expecting udp:<hostname>:<port> or unix:<path-to-socket> format. Got: \"\"",
		},
		{
			name:        "More than one separator",
			teststring:  "a:b:c",
			wantNetwork: "a",
			wantAddress: "b:c",
		},
		{
			name:        "Many separators",
			teststring:  ":::::",
			wantNetwork: "",
			wantAddress: "::::",
		},
		{
			name:        "One separator only",
			teststring:  ":",
			wantNetwork: "",
			wantAddress: "",
		},
		{
			name:        "Convert Upper Case network to lower case but leave address alone",
			teststring:  "FOO:BAR",
			wantNetwork: "foo",
			wantAddress: "BAR",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &statsd{}
			network, address, err := s.parseSocketEnv(tt.teststring)

			utilerror.AssertErrorMessage(t, err, tt.wantError)

			if network != tt.wantNetwork {
				t.Error(network)
			}
			if address != tt.wantAddress {
				t.Error(address)
			}
		})
	}
}

func TestValidateSocketDefinition(t *testing.T) {
	for _, tt := range []struct {
		name         string
		network      string
		address      string
		expectToPass bool
		wantError    string
	}{
		{
			name:         "Valid UDP case",
			network:      "udp",
			address:      "127.0.0.1:9000",
			expectToPass: true,
		},
		{
			name:         "Valid Unix Domain Socket case",
			network:      "unix",
			address:      "/var/something/or/another",
			expectToPass: true,
		},
		{
			name:         "Invalid protocoll",
			network:      "tcp",
			address:      "127.0.0.1:12",
			expectToPass: false,
			wantError:    "unsupported protocol for the mdm statds socket. Expecting  'udp:' or 'unix:'. Got: \"tcp\"",
		},
		{
			name:         "Valid protocol, random invalid address - is good here",
			network:      "udp",
			address:      "somerandomtext149203#$%",
			expectToPass: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &statsd{}

			ok, err := s.validateSocketDefinition(tt.network, tt.address)

			utilerror.AssertErrorMessage(t, err, tt.wantError)

			if ok != tt.expectToPass {
				t.Error(ok)
			}
		})
	}
}

func TestConnectionDetails(t *testing.T) {
	for _, tt := range []struct {
		name string

		isLocalDev      bool
		mdmsocketstring string
		network         string
		address         string
		wantError       string
	}{
		{
			name:            "Old behaviour / production mode",
			isLocalDev:      false,
			mdmsocketstring: "",
			network:         "unix",
			address:         "/var/etw/mdm_statsd.socket",
		},
		{
			name:            "Old Behaviour / localdev mode",
			isLocalDev:      true,
			mdmsocketstring: "",
			network:         "unix",
			address:         "mdm_statsd.socket",
		},
		{
			name:            "Valid UDP env variable",
			isLocalDev:      false,
			mdmsocketstring: "udp:127.0.0.1:9000",
			network:         "udp",
			address:         "127.0.0.1:9000",
		},
		{
			name:            "Don't override default in localdev ",
			isLocalDev:      true,
			mdmsocketstring: "udp:127.0.0.1:9000",
			network:         "udp",
			address:         "127.0.0.1:9000",
		},
		{
			name:            "Random string without separator",
			isLocalDev:      true,
			mdmsocketstring: "another random string without separator",
			wantError:       "malformed definition for the mdm statds socket. Expecting udp:<hostname>:<port> or unix:<path-to-socket> format. Got: \"another random string without separator\"",
		},
		{
			name:            "Invalid Protocol",
			isLocalDev:      true,
			mdmsocketstring: "foo:bar",
			wantError:       "unsupported protocol for the mdm statds socket. Expecting  'udp:' or 'unix:'. Got: \"foo\"",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			env := mock_env.NewMockInterface(controller)

			if tt.mdmsocketstring == "" {
				if tt.isLocalDev {
					env.EXPECT().IsLocalDevelopmentMode().Return(true)
				} else {
					env.EXPECT().IsLocalDevelopmentMode().Return(false)
				}
			}

			s := &statsd{
				env:          env,
				mdmSocketEnv: tt.mdmsocketstring,
			}

			network, address, err := s.connectionDetails()

			utilerror.AssertErrorMessage(t, err, tt.wantError)

			if network != tt.network {
				t.Error(network)
			}
			if address != tt.address {
				t.Error(address)
			}
		})
	}
}
