package clientauthorizer

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestARMIsReady(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	fakeNow, _ := time.Parse(time.RFC3339, "2020-01-20T00:00:00Z")
	fakeNowFunc := func() time.Time { return fakeNow }

	tests := []struct {
		name    string
		do      func(req *http.Request) (*http.Response, error)
		wantErr string
	}{
		{
			name: "success",
			do: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: ioutil.NopCloser(strings.NewReader(
						`{
							"clientCertificates": [
								{
									"notBefore": "2020-01-19T23:00:00Z",
									"notAfter": "2020-01-20T01:00:00Z",
									"certificate": "dGVzdA=="
								}
							]
						}`,
					)),
				}, nil
			},
		},
		{
			name: "invalid JSON",
			do: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: ioutil.NopCloser(strings.NewReader("not JSON")),
				}, nil
			},
			wantErr: "invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name: "request - error",
			do: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("fake error")
			},
			wantErr: "fake error",
		},
		{
			name: "request - unexpected status code",
			do: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: ioutil.NopCloser(strings.NewReader("{}")),
				}, nil
			},
			wantErr: "unexpected status code 502",
		},
		{
			name: "request - unexpected content type",
			do: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
					Body: ioutil.NopCloser(strings.NewReader("")),
				}, nil
			},
			wantErr: `unexpected content type "text/plain"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := &arm{
				log: log,
				now: fakeNowFunc,
				do:  test.do,
			}

			if a.IsReady() == true {
				t.Fatal("unexpected ready state")
			}

			// Ignore possible errors. If an error occurs, authoriser must not be ready.
			// We are testing this behaviour via checking result of a.IsReady()
			err := a.refreshOnce()
			if err != nil && err.Error() != test.wantErr {
				t.Errorf("got error %#v, expected %#v", err.Error(), test.wantErr)
			}

			if test.wantErr != "" && a.IsReady() {
				t.Fatal("unexpected ready state")
			}
		})
	}
}

func TestARMIsAuthorized(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	fakeNow, _ := time.Parse(time.RFC3339, "2020-01-20T00:00:00Z")
	fakeNowFunc := func() time.Time { return fakeNow }

	tests := []struct {
		name string
		m    metadata
		cs   *tls.ConnectionState
		want bool
	}{
		{
			name: "cert is present in the chain - single client cert in meta",
			m: metadata{ClientCertificates: []clientCertificate{
				{
					Thumbprint:  "current",
					Certificate: []byte("current"),
					NotBefore:   fakeNow.Add(-time.Hour),
					NotAfter:    fakeNow.Add(time.Hour),
				},
			}},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{Raw: []byte("current")},
				},
			},
			want: true,
		},
		{
			name: "cert is present in the chain - multiple client certs in meta",
			m: metadata{ClientCertificates: []clientCertificate{
				{
					Thumbprint:  "past",
					Certificate: []byte("past"),
					NotBefore:   fakeNow.Add(-6 * time.Hour),
					NotAfter:    fakeNow.Add(-5 * time.Hour),
				},
				{
					Thumbprint:  "current",
					Certificate: []byte("current"),
					NotBefore:   fakeNow.Add(-time.Hour),
					NotAfter:    fakeNow.Add(time.Hour),
				},
				{
					Thumbprint:  "future",
					Certificate: []byte("future"),
					NotBefore:   fakeNow.Add(5 * time.Hour),
					NotAfter:    fakeNow.Add(6 * time.Hour),
				},
			}},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{Raw: []byte("current")},
				},
			},
			want: true,
		},
		{
			name: "cert is present in the chain - not latest",
			m: metadata{ClientCertificates: []clientCertificate{
				{
					Thumbprint:  "current",
					Certificate: []byte("current"),
					NotBefore:   fakeNow.Add(-time.Hour),
					NotAfter:    fakeNow.Add(time.Hour),
				},
			}},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{Raw: []byte("does not match")},
					{Raw: []byte("current")},
				},
			},
		},
		{
			name: "invalid connection state - empty",
		},
		{
			name: "invalid connection state - no PeerCertificates",
			cs:   &tls.ConnectionState{ServerName: "test"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := &arm{
				log:                   log,
				now:                   fakeNowFunc,
				m:                     test.m,
				lastSuccessfulRefresh: fakeNow,
			}

			if !a.IsReady() {
				t.Fatal("expected ready state")
			}

			result := a.IsAuthorized(test.cs)
			if result != test.want {
				t.Fatalf("got %#v, expected %#v", result, test.want)
			}
		})
	}
}
