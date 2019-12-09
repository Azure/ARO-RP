package clientauthorizer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/util/recover"
)

type metadata struct {
	ClientCertificates []struct {
		Thumbprint  string    `json:"thumbprint,omitempty"`
		NotBefore   time.Time `json:"notBefore,omitempty"`
		NotAfter    time.Time `json:"notAfter,omitempty"`
		Certificate []byte    `json:"certificate,omitempty"`
	} `json:"clientCertificates,omitempty"`
}

type arm struct {
	log *logrus.Entry
	now func() time.Time
	do  func(*http.Request) (*http.Response, error)

	mu sync.RWMutex
	m  metadata

	lastSuccessfulRefresh time.Time
}

func NewARM(log *logrus.Entry) ClientAuthorizer {
	a := &arm{
		log: log,
		now: time.Now,
		do:  http.DefaultClient.Do,
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

	req, err := http.NewRequest(http.MethodGet, "https://management.azure.com:24582/metadata/authentication?api-version=2015-01-01", nil)
	if err != nil {
		return err
	}

	resp, err := a.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %q", resp.StatusCode)
	}

	if strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0] != "application/json" {
		return fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	var m *metadata
	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.m = *m
	a.lastSuccessfulRefresh = now

	return nil
}

func (a *arm) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.now().Add(-6 * time.Hour).Before(a.lastSuccessfulRefresh)
}
