package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

type server struct {
	containerClient Client
	verboseLogging  bool
}

type errTokenValidation struct {
	StatusCode       int
	ErrorDescription []string
	WWWAuthenticate  []string
}

type arm struct {
	log *logrus.Entry
	im  instancemetadata.InstanceMetadata
	now func() time.Time
	do  func(*http.Request) (*http.Response, error)

	mu sync.RWMutex
	m  metadata

	lastSuccessfulRefresh time.Time

	s server
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
		s: server{
			containerClient: NewMise(http.DefaultClient, "http://localhost:5000/ValidateRequest"),
			verboseLogging:  false,
		},
	}

	go a.refresh()

	return a
}

func (a *arm) IsAuthorized(r *http.Request) bool {

	if a.authenticatedHandler(r) {
		return true
	}

	cs := r.TLS
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

func (e *errTokenValidation) Error() string {
	return fmt.Sprintf("StatusCode: %d, ErrorDescription: %v, WWWAuthenticate: %v", e.StatusCode, e.ErrorDescription, e.WWWAuthenticate)
}

func (a *arm) delegateAuthToContainer(authHeader, uri, method, ipAddr string) (Result, error) {
	if a.s.verboseLogging {
		a.log.Printf(
			"VERBOSE: Original request Information:\nURL:%s, method:%s, original IP address:%s",
			uri,
			method,
			ipAddr,
		)
	}

	start := time.Now()

	result, err := a.s.containerClient.ValidateRequest(context.Background(), Input{
		OriginalUri:         uri,
		OriginalMethod:      method,
		OriginalIPAddress:   ipAddr,
		AuthorizationHeader: authHeader,
		// Replace SubjectClaimsToReturn with ReturnAllSubjectClaims
		// to return all claims in the subject token instead of just an allow list.
		ReturnAllSubjectClaims: true,
		//SubjectClaimsToReturn: []string{"Preferred_username"},
	})

	end := time.Now()
	elapsed := end.Sub(start)

	log.Default().Printf("time elapsed in mise container + adapter: %d", elapsed.Milliseconds())

	if err != nil {
		return Result{}, fmt.Errorf("error while validating token: %w", err)
	}

	if a.s.verboseLogging {
		json, err := json.MarshalIndent(result, "", "   ")
		if err != nil {
			log.Default().Printf("error marshalling json of result object err=%v\n", err)
		} else {
			log.Default().Printf("VERBOSE: result struct:\n%s", string(json))
		}
	}

	if result.StatusCode == http.StatusOK {
		return result, nil
	} else {
		return Result{}, &errTokenValidation{
			StatusCode:       result.StatusCode,
			ErrorDescription: result.ErrorDescription,
			WWWAuthenticate:  result.WWWAuthenticate,
		}
	}
}

func (a *arm) authenticatedHandler(r *http.Request) bool {
	authHeaders := r.Header["Authorization"]

	if len(authHeaders) != 1 {
		return false
	}

	authHeader := authHeaders[0]
	if authHeader == "" {
		//w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	fullUriString := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())

	_, err := a.delegateAuthToContainer(authHeader, fullUriString, r.Method, r.RemoteAddr)
	if err != nil {
		var validationErr *errTokenValidation
		if errors.As(err, &validationErr) {
			// can access validationErr.ErrorDescription, validationErr.WWWAuthenticate, validationErr.StatusCode
			a.log.Printf("token validation err:%v", validationErr)
		} else {
			a.log.Printf("error while delegating auth:%v", err)
		}

		return false
	}

	return true
}
