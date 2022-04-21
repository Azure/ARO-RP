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

	os.Setenv(STATSD_SOCKET_ENV, "udp:"+socket)
	c2, _ := net.ListenPacket("udp", socket)
	//c2.SetReadDeadline(time.Now().Add(5 * time.Second))

	s := &statsd{
		env: env,

		account:   "*",
		namespace: "*",

		ch: make(chan *metric),

		now: func() time.Time { return time.Time{} },
	}

	c := make(chan string)
	go func() {
		buf := make([]byte, 1024)
		// set 5 second read timeout. That should be plenty of time to receive the emitted Gauge
		c2.SetReadDeadline(time.Now().Add(5 * time.Second))
		m := ""
		//fmt.Println("Read")
		n, _, err := c2.ReadFrom(buf)
		if err != nil {
			//fmt.Println(err.Error())
			//m = err.Error()
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
		name             string
		set              bool
		isLocalDev       bool
		socketstring     string
		protocol         string
		connectionstring string
		wantError        bool
	}{
		{
			name:         "Random string",
			set:          true,
			socketstring: "randomstring234234809$#54ew5",
			wantError:    true,
		},
		{
			name:             "Old Bevaviour / production mode",
			set:              false,
			protocol:         "unix",
			connectionstring: "/var/etw/mdm_statsd.socket",
			wantError:        false,
		},
		{
			name:             "Old Bevaviour / localdev mode",
			set:              false,
			isLocalDev:       true,
			protocol:         "unix",
			connectionstring: "mdm_statsd.socket",
			wantError:        false,
		},
		{
			name:         "Empty env variable",
			set:          true,
			socketstring: "",
			wantError:    true,
		},
		{
			name:             "Valid UDP env variable",
			set:              true,
			socketstring:     "udp:127.0.0.1:9000",
			protocol:         "udp",
			connectionstring: "127.0.0.1:9000",
			wantError:        false,
		},
		{
			name:             "Valid Unix domain socket env variable",
			set:              true,
			socketstring:     "unix:test.socket",
			protocol:         "unix",
			connectionstring: "test.socket",
			wantError:        false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
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

			os.Unsetenv(STATSD_SOCKET_ENV)
			if tt.set {
				os.Setenv(STATSD_SOCKET_ENV, tt.socketstring)
			}

			p, c, err := s.getConnectionDetails()
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
				} else if p != tt.protocol {
					t.Fail()
				} else if c != tt.connectionstring {
					t.Fail()
				}

			}
		})

	}

}
