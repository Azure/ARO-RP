package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

var (
	serverkey, clientkey     *rsa.PrivateKey
	servercerts, clientcerts []*x509.Certificate
)

func init() {
	var err error

	clientkey, clientcerts, err = utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		panic(err)
	}

	serverkey, servercerts, err = utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		panic(err)
	}
}

type testInfra struct {
	env        env.Interface
	controller *gomock.Controller
	l          net.Listener
	cli        *http.Client
}

func newTestInfra(t *testing.T) (*testInfra, error) {
	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	l := listener.NewListener()

	env := &env.Test{
		TestLocation: "eastus",
	}

	return &testInfra{
		env:        env,
		controller: gomock.NewController(t),
		l:          TLSListener(l, serverkey, servercerts),
		cli: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
					Certificates: []tls.Certificate{
						{
							Certificate: [][]byte{clientcerts[0].Raw},
							PrivateKey:  clientkey,
						},
					},
				},
				Dial: l.Dial,
			},
		},
	}, nil
}

func (ti *testInfra) done() error {
	ti.controller.Finish()
	ti.cli.CloseIdleConnections()
	return ti.l.Close()
}

func (ti *testInfra) request(method, url string, header http.Header, in interface{}) (*http.Response, []byte, error) {
	var b []byte

	if in != nil {
		var err error
		b, err = json.Marshal(in)
		if err != nil {
			return nil, nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}

	req.Header = header

	resp, err := ti.cli.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, b, nil
}

func validateResponse(resp *http.Response, b []byte, wantStatusCode int, wantError string, wantResponse interface{}) error {
	if resp.StatusCode != wantStatusCode {
		return fmt.Errorf("unexpected status code %d, wanted %d", resp.StatusCode, wantStatusCode)
	}

	if wantError != "" {
		cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
		err := json.Unmarshal(b, &cloudErr)
		if err != nil {
			return err
		}

		if cloudErr.Error() != wantError {
			return fmt.Errorf("unexpected error %s, wanted %s", cloudErr.Error(), wantError)
		}

		return nil
	}

	if wantResponse == nil || reflect.ValueOf(wantResponse).IsZero() {
		if len(b) != 0 {
			return fmt.Errorf("unexpected response %s, wanted no content", string(b))
		}
		return nil
	}

	if wantResponse, ok := wantResponse.([]byte); ok {
		if !bytes.Equal(b, wantResponse) {
			return fmt.Errorf("unexpected response %s, wanted %s", string(b), string(wantResponse))
		}
		return nil
	}

	v := reflect.New(reflect.TypeOf(wantResponse).Elem()).Interface()
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(v, wantResponse) {
		return fmt.Errorf("unexpected response %s, wanted to match %#v", string(b), wantResponse)
	}

	return nil
}
