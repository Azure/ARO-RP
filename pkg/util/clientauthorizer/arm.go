package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type metadata struct {
	ClientCertificates []clientCertificate `json:"clientCertificates,omitempty"`
}

type clientCertificate struct {
	Thumbprint  string    `json:"thumbprint,omitempty"`
	NotBefore   time.Time `json:"notBefore,omitempty"`
	NotAfter    time.Time `json:"notAfter,omitempty"`
	Certificate []byte    `json:"certificate,omitempty"`
}

type arm struct {
	log *logrus.Entry
	im  instancemetadata.InstanceMetadata
	now func() time.Time
	do  func(*http.Request) (*http.Response, error)

	mu sync.RWMutex
	m  metadata

	lastSuccessfulRefresh time.Time
}

func NewARM(log *logrus.Entry, im instancemetadata.InstanceMetadata) ClientAuthorizer {
	c := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("tried to redirect")
		},
	}

	a := &arm{
		log: log,
		im:  im,
		now: time.Now,
		do:  c.Do,
	}

	go a.refresh()

	return a
}

func (a *arm) IsAuthorized(cs *tls.ConnectionState) bool {
	if cs == nil || len(cs.PeerCertificates) == 0 {
		return false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	now := a.now()
	for _, c := range a.m.ClientCertificates {
		if c.NotBefore.Before(now) &&
			c.NotAfter.After(now) &&
			bytes.Equal(c.Certificate, cs.PeerCertificates[0].Raw) {
			return true
		}
	}

	return false
}

func (a *arm) refresh() {
	defer recover.Panic(a.log)

	t := time.NewTicker(time.Hour)

	for {
		a.log.Print("refreshing metadata")

		err := a.refreshOnce()
		if err != nil {
			a.log.Error(err)
		}

		<-t.C
	}
}

func (a *arm) refreshOnce() error {
	now := a.now()

	// ARM <-> RP Authentication endpoint is not consistent.  Check ARM wiki for up-to-date metadata endpoints
	endpoint := strings.TrimSuffix(a.im.Environment().ResourceManagerEndpoint, "/") + ":24582"
	if reflect.DeepEqual(a.im.Environment().Environment, azure.PublicCloud) {
		endpoint = "https://admin.management.azure.com"
	}

	req, err := http.NewRequest(http.MethodGet, endpoint+"/metadata/authentication?api-version=2015-01-01", nil)
	if err != nil {
		return err
	}

	resp, err := a.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	if strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0] != "application/json" {
		return fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	var m *metadata
	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return err
	}

	var ok bool
	for _, c := range m.ClientCertificates {
		if c.NotBefore.Before(now) &&
			c.NotAfter.After(now) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("did not receive current certificate")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.m = *m
	a.lastSuccessfulRefresh = now

	return nil
}

func (a *arm) IsReady() bool {
	now := a.now()

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.lastSuccessfulRefresh.Add(24 * time.Hour).Before(a.now()) {
		return false
	}

	for _, c := range a.m.ClientCertificates {
		if c.NotBefore.Before(now) &&
			c.NotAfter.After(now) {
			return true
		}
	}

	return false
}
