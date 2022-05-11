package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
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
		name       string
		teststring string
		part1      string
		part2      string
		wantError  bool
	}{
		{
			name:       "Valid string",
			teststring: "part1:part2",
			part1:      "part1",
			part2:      "part2",
			wantError:  false,
		},
		{
			name:       "Empty first part-regarded valid",
			teststring: ":part2",
			part1:      "",
			part2:      "part2",
			wantError:  false,
		},
		{
			name:       "Empty second part-regarded valid",
			teststring: "part1:",
			part1:      "part1",
			part2:      "",
			wantError:  false,
		},
		{
			name:       "No separator",
			teststring: "somerandommtext",
			wantError:  true,
		},
		{
			name:       "Empty string",
			teststring: "",
			wantError:  true,
		},
		{
			name:       "More than one separator",
			teststring: "a:b:c",
			part1:      "a",
			part2:      "b:c",
			wantError:  false,
		},
		{
			name:       "Many separators",
			teststring: ":::::",
			part1:      "",
			part2:      "::::",
			wantError:  false,
		},
		{
			name:       "One separator only",
			teststring: ":",
			part1:      "",
			part2:      "",
			wantError:  false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &statsd{}
			part1, part2, err := s.parseSocketEnv(tt.teststring)

			if tt.wantError {
				if err == nil {
					t.Error(fmt.Errorf("expected error but didn't get one."))
				}
			} else {
				if err != nil {
					t.Error(fmt.Errorf("unexpected error received: %q ", err.Error()))
				}
				if part1 != tt.part1 {
					t.Error(fmt.Errorf("part1 does not match: Wanted: %q but got %q ", tt.part1, part1))
				}
				if part2 != tt.part2 {
					t.Error(fmt.Errorf("part2 does not match: Wanted: %q but got %q ", tt.part2, part2))
				}
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
		wantError    bool
	}{
		{
			name:         "Valid UDP case",
			network:      "udp",
			address:      "127.0.0.1:9000",
			expectToPass: true,
			wantError:    false,
		},
		{
			name:         "Valid Unix Domain Socket case",
			network:      "unix",
			address:      "/var/something/or/another",
			expectToPass: true,
			wantError:    false,
		},
		{
			name:         "Invalid protocoll",
			network:      "tcp",
			address:      "127.0.0.1:12",
			expectToPass: false,
			wantError:    true,
		},
		{
			name:         "Valid protocol, random invalid address - is good here",
			network:      "udp",
			address:      "somerandomtext149203#$%",
			expectToPass: true,
			wantError:    false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &statsd{}

			ok, err := s.validateSocketDefinition(tt.network, tt.address)

			if tt.wantError && err == nil {
				t.Error(fmt.Errorf("Test %s, expected error but didn't get one.", tt.name))
			}

			if !tt.wantError && err != nil {
				t.Error(fmt.Errorf("Test %s, unexpected validation error %q.", tt.name, err.Error()))
			}

			if ok != tt.expectToPass {
				t.Error(fmt.Errorf("Test %q,unexpected validation result: Expected: %t, Got %t ", tt.name, tt.expectToPass, ok))
			}
		})
	}
}

func TestGetConnectionDetails(t *testing.T) {
	for _, tt := range []struct {
		name string

		isLocalDev           bool
		mdmsocketstring      string
		network              string
		address              string
		wantErrorToStartWith string
	}{
		{
			name:                 "Old behaviour / production mode",
			isLocalDev:           false,
			mdmsocketstring:      "",
			network:              "unix",
			address:              "/var/etw/mdm_statsd.socket",
			wantErrorToStartWith: "",
		},
		{
			name:                 "Old Behaviour / localdev mode",
			isLocalDev:           true,
			mdmsocketstring:      "",
			network:              "unix",
			address:              "mdm_statsd.socket",
			wantErrorToStartWith: "",
		},
		{
			name:                 "Valid UDP env variable",
			isLocalDev:           false,
			mdmsocketstring:      "udp:127.0.0.1:9000",
			network:              "udp",
			address:              "127.0.0.1:9000",
			wantErrorToStartWith: "",
		},
		{
			name:                 "Don't override default in localdev ",
			isLocalDev:           true,
			mdmsocketstring:      "udp:127.0.0.1:9000",
			network:              "udp",
			address:              "127.0.0.1:9000",
			wantErrorToStartWith: "",
		},
		{
			name:                 "Random string without separator",
			isLocalDev:           true,
			mdmsocketstring:      "another random string without separator",
			wantErrorToStartWith: "malformed definition",
		},
		{
			name:                 "Invalid Protocol",
			isLocalDev:           true,
			mdmsocketstring:      "foo:bar",
			wantErrorToStartWith: "unsupported protocol",
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
				mdmsocketEnv: tt.mdmsocketstring,
			}

			network, address, err := s.getConnectionDetails()

			if tt.wantErrorToStartWith != "" {
				if err == nil {
					t.Error(fmt.Errorf("Test %s, expected error \"%s...\" but didn't get one.", tt.name, tt.wantErrorToStartWith))
				} else if !strings.HasPrefix(err.Error(), tt.wantErrorToStartWith) {
					t.Error(fmt.Errorf("Test %s, unexpected error received. Expected \"%s...\" but got %q.", tt.name, tt.wantErrorToStartWith, err.Error()))
				}
			} else {
				if err != nil {
					t.Error(fmt.Errorf("Test %q,unexpected error received: %q ", tt.name, err.Error()))
				}
				if network != tt.network {
					t.Error(fmt.Errorf("Test %q,network does not match: Wanted: %q but got %q ", tt.name, tt.network, network))
				}
				if address != tt.address {
					t.Error(fmt.Errorf("Test %q,part2 does not match: Wanted: %q but got %q ", tt.name, tt.address, address))
				}
			}
		})
	}
}
