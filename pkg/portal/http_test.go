package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	frontendmiddleware "github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type testPortal struct {
	p             *portal
	l             *listener.Listener
	auditHook     *test.Hook
	portalLogHook *test.Hook
}

func NewTestPortal(_env env.Core, dbOpenShiftClusters database.OpenShiftClusters, dbPortal database.Portal) *testPortal {
	_, portalAccessLog := testlog.New()
	portalLogHook, portalLog := testlog.New()
	auditHook, portalAuditLog := testlog.NewAudit()

	l := listener.NewListener()
	p := NewPortal(_env, portalAuditLog, portalLog, portalAccessLog, l, nil, nil, "", nil, nil, "", nil, nil, make([]byte, 32), nil, nonElevatedGroupIDs, elevatedGroupIDs, dbOpenShiftClusters, dbPortal, nil, nil).(*portal)

	return &testPortal{
		p:             p,
		l:             l,
		auditHook:     auditHook,
		portalLogHook: portalLogHook,
	}
}

func (p *testPortal) DumpLogs(t *testing.T) {
	for _, l := range p.portalLogHook.Entries {
		t.Error(l)
	}
}

func (p *testPortal) Run(ctx context.Context) error {
	router, err := p.p.setupRouter(nil, nil, nil)
	if err != nil {
		return err
	}

	s := &http.Server{
		Handler:     frontendmiddleware.Lowercase(router),
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    log.New(p.p.log.Writer(), "", 0),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	go func() {
		err := s.Serve(p.l)
		if err != nil {
			p.p.log.Error(err)
		}
	}()

	return nil
}

func (p *testPortal) Request(method string, path string, authenticated bool, elevated bool) (*http.Response, error) {
	p.portalLogHook.Reset()

	req, err := http.NewRequest(method, "http://server"+path, nil)
	if err != nil {
		return nil, err
	}

	err = addCSRF(req)
	if err != nil {
		return nil, err
	}

	if authenticated {
		var groups []string
		if elevated {
			groups = elevatedGroupIDs
		} else {
			groups = nonElevatedGroupIDs
		}
		err = addAuth(req, groups)
		if err != nil {
			return nil, err
		}
	}

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: p.p.l.(*listener.Listener).DialContext,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (p *testPortal) Cleanup() {
	p.l.Close()
}
