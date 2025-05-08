package clientcache

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"testing"
	"time"
)

func TestClientCache(t *testing.T) {
	var now time.Time

	cli1 := &http.Client{}
	cli2 := &http.Client{}

	c := New(1)
	c.(*clientCache).now = func() time.Time { return now }

	// t = 0: put(1), get(1)
	c.Put(1, cli1)
	if c.Get(1) != cli1 {
		t.Error(c.Get(1), cli1)
	}

	now = now.Add(2)

	// t = 2: put(1), get(1) (cli1's ttl should be reset before expiring)
	if c.Get(1) != cli1 {
		t.Error(c.Get(1), cli1)
	}

	now = now.Add(2)

	// t = 4: put(2), get(2) (cli1 should be expired)
	c.Put(2, cli2)
	if c.Get(2) != cli2 {
		t.Error(c.Get(2), cli2)
	}

	if c.Get(1) != nil {
		t.Error(c.Get(1))
	}

	now = now.Add(2)

	// t = 6: cli2 should be expired
	c.(*clientCache).expire()

	if c.Get(2) != nil {
		t.Error(c.Get(2))
	}
}
