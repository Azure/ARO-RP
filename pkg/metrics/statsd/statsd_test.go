package statsd

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
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

func TestEmitGaugeViaUDP(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	env := mock_env.NewMockInterface(controller)
	env.EXPECT().Location().AnyTimes().Return("eastus")
	env.EXPECT().Hostname().AnyTimes().Return("test-host")

	socket := "127.0.0.1:8001"

	os.Setenv(statsdSocketEnv, "udp:"+socket)
	c2, _ := net.ListenPacket("udp", socket)

	s := &statsd{
		env:       env,
		account:   "*",
		namespace: "*",
		ch:        make(chan *metric),
		now:       func() time.Time { return time.Time{} },
	}

	c := make(chan string)
	go func() {
		buf := make([]byte, 1024)
		// set 5 second read timeout. That should be plenty of time to receive the emitted Gauge
		c2.SetReadDeadline(time.Now().Add(5 * time.Second))
		m := ""
		n, _, err := c2.ReadFrom(buf)
		if err != nil {
			m = err.Error()
		} else {
			m = string(buf[:n])
		}

		c <- m
	}()

	go s.run()
	s.EmitGauge("tests.test_key", 42, map[string]string{"key": "value"})
	m := <-c
	if m != `{"Metric":"tests.test_key","Account":"*","Namespace":"*","Dims":{"hostname":"test-host","key":"value","location":"eastus"},"TS":"0001-01-01T00:00:00.000"}:42|g`+"\n" {
		t.Error(errors.New(m))
	}
}

func TestGetConnectionDetails(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name         string
		set          bool
		isLocalDev   bool
		socketstring string
		network      string
		address      string
		wantError    bool
	}{
		{
			name:      "Old Bevaviour / production mode",
			set:       false,
			network:   "unix",
			address:   "/var/etw/mdm_statsd.socket",
			wantError: false,
		},
		{
			name:       "Old Bevaviour / localdev mode",
			set:        false,
			isLocalDev: true,
			network:    "unix",
			address:    "mdm_statsd.socket",
			wantError:  false,
		},
		{
			name:         "Valid UDP env variable",
			set:          true,
			socketstring: "udp:127.0.0.1:9000",
			network:      "udp",
			address:      "127.0.0.1:9000",
			wantError:    false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := mock_env.NewMockInterface(controller)

			if !tt.set {
				if tt.isLocalDev {
					env.EXPECT().IsLocalDevelopmentMode().Return(true)
				} else {
					env.EXPECT().IsLocalDevelopmentMode().Return(false)
				}
			}

			s := &statsd{
				env: env,
			}

			os.Unsetenv(statsdSocketEnv)
			if tt.set {
				os.Setenv(statsdSocketEnv, tt.socketstring)
			}

			network, address, err := s.getConnectionDetails()
			if err != nil {
				fmt.Println(err.Error())
			}

			if tt.wantError {
				if err == nil {
					t.Fail()
				}
			} else {
				if err != nil && tt.wantError == false {
					fmt.Println(err.Error())
					t.Fail()
				}
				if network != tt.network {
					t.Fail()
				}
				if address != tt.address {
					t.Fail()
				}
			}
		})
	}
}

func TestParseSocketEnv(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	for _, tt := range []struct {
		name      string
		socketEnv string
		network   string
		address   string
		wantError bool
	}{
		{
			name:      "Random string",
			socketEnv: "randomstring234234809$#54ew5",
			wantError: true,
		},
		{
			name:      "Empty env variable",
			socketEnv: "",
			wantError: true,
		},
		{
			name:      "Valid UDP env variable",
			socketEnv: "udp:127.0.0.1:9000",
			network:   "udp",
			address:   "127.0.0.1:9000",
			wantError: false,
		},
		{
			name:      "Valid UDP env variable",
			socketEnv: "UDP:127.0.0.1:9000",
			network:   "udp",
			address:   "127.0.0.1:9000",
			wantError: false,
		},
		{
			name:      "Valid UDP DNS env variable",
			socketEnv: "udp:localhost:9000",
			network:   "udp",
			address:   "localhost:9000",
			wantError: false,
		},
		{
			name:      "Valid Unix domain socket env variable",
			socketEnv: "unix:test.socket",
			network:   "unix",
			address:   "test.socket",
			wantError: false,
		},
		{
			name:      "Unsupported /network",
			socketEnv: "tcp:127.0.0.1:8125",
			network:   "",
			address:   "",
			wantError: true,
		},
		{
			name:      "Invalid UDP env variable",
			socketEnv: "udp:127.0.0.1:90000000",
			network:   "",
			address:   "",
			wantError: true,
		},
		{
			name:      "Valid IPV6 UDP env variable",
			socketEnv: "udp:[2001:db8:3333:4444:5555:6666:7777:8888]:9000",
			network:   "udp",
			address:   "[2001:db8:3333:4444:5555:6666:7777:8888]:9000",
			wantError: false,
		},
		{
			name:      "Valid Unix Test",
			socketEnv: "unix:/var/this/that/or/another",
			network:   "unix",
			address:   "/var/this/that/or/another",
			wantError: false,
		},
		{
			name:      "Valid Unix Path",
			socketEnv: "unix:../another.test.socket",
			network:   "unix",
			address:   "../another.test.socket",
			wantError: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := mock_env.NewMockInterface(controller)

			s := &statsd{
				env: env,
			}

			network, address, err := s.parseSocketEnv(tt.socketEnv)
			if err != nil {
				fmt.Println(err.Error())
			}

			if tt.wantError {
				if err == nil {
					t.Fail()
				}
			} else {
				if err != nil && tt.wantError == false {
					fmt.Println(err.Error())
					t.Fail()
				}
				if network != tt.network {
					t.Fail()
				}
				if address != tt.address {
					t.Fail()
				}
			}
		})
	}
}
