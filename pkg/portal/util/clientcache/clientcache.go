package clientcache

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"sync"
	"time"
)

// ClientCache is a cache for *http.Clients.  It allows us to reuse clients and
// connections across multiple incoming calls, saving us TCP, TLS and proxy
// initialisations.
type ClientCache interface {
	Get(interface{}) *http.Client
	Put(interface{}, *http.Client)
}

type clientCache struct {
	mu  sync.Mutex
	now func() time.Time
	ttl time.Duration
	m   map[interface{}]*v
}

type v struct {
	expires time.Time
	cli     *http.Client
}

// New returns a new ClientCache
func New(ttl time.Duration) ClientCache {
	return &clientCache{
		now: time.Now,
		ttl: ttl,
		m:   map[interface{}]*v{},
	}
}

// call holding c.mu
func (c *clientCache) expire() {
	now := c.now()
	for k, v := range c.m {
		if now.After(v.expires) {
			v.cli.CloseIdleConnections()
			delete(c.m, k)
		}
	}
}

func (c *clientCache) Get(k interface{}) (cli *http.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v := c.m[k]; v != nil {
		v.expires = c.now().Add(c.ttl)
		cli = v.cli
	}

	c.expire()

	return
}

func (c *clientCache) Put(k interface{}, cli *http.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[k] = &v{
		expires: c.now().Add(c.ttl),
		cli:     cli,
	}

	c.expire()
}
