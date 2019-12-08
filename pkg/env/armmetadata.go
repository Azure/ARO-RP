package env

import (
	"bytes"
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

type armMetadataService struct {
	log *logrus.Entry

	mu sync.RWMutex
	m  metadata

	lastSuccessfulRefresh time.Time
}

func newARMMetadataService(log *logrus.Entry) *armMetadataService {
	ms := &armMetadataService{log: log}

	go ms.refresh()

	return ms
}

func (ms *armMetadataService) allowClientCertificate(rawCert []byte) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	now := time.Now()
	for _, c := range ms.m.ClientCertificates {
		if c.NotBefore.Before(now) &&
			c.NotAfter.After(now) &&
			bytes.Equal(c.Certificate, rawCert) {
			return true
		}
	}

	return false
}

func (ms *armMetadataService) refresh() {
	defer recover.Panic(ms.log)

	t := time.NewTicker(time.Hour)

	for {
		ms.log.Print("refreshing metadata")
		err := ms.refreshOnce()
		if err != nil {
			ms.log.Error(err)
		}

		<-t.C
	}
}

func (ms *armMetadataService) refreshOnce() error {
	now := time.Now()

	resp, err := http.Get("https://management.azure.com:24582/metadata/authentication?api-version=2015-01-01")
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

	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.m = *m
	ms.lastSuccessfulRefresh = now

	return nil
}

func (ms *armMetadataService) isReady() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return time.Now().Add(-6 * time.Hour).Before(ms.lastSuccessfulRefresh)
}
