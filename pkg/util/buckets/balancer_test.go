package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-RP/pkg/api"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestBalance(t *testing.T) {
	type test struct {
		name     string
		monitors []string
		doc      func() *api.PoolWorkerDocument
		validate func(*testing.T, *test, *api.PoolWorkerDocument)
	}

	for _, tt := range []*test{
		{
			name:     "0->1",
			monitors: []string{"one"},
			doc: func() *api.PoolWorkerDocument {
				return &api.PoolWorkerDocument{}
			},
			validate: func(t *testing.T, tt *test, doc *api.PoolWorkerDocument) {
				for i, bucket := range doc.PoolWorker.Buckets {
					if bucket != "one" {
						t.Error(i, bucket)
					}
				}
			},
		},
		{
			name:     "3->1",
			monitors: []string{"one"},
			doc: func() *api.PoolWorkerDocument {
				return &api.PoolWorkerDocument{
					PoolWorker: &api.PoolWorker{
						Buckets: []string{"one", "two", "one", "three", "one", "two", "two", "one", "two"},
					},
				}
			},
			validate: func(t *testing.T, tt *test, doc *api.PoolWorkerDocument) {
				for i, bucket := range doc.PoolWorker.Buckets {
					if bucket != "one" {
						t.Error(i, bucket)
					}
				}
			},
		},
		{
			name: "3->0",
			doc: func() *api.PoolWorkerDocument {
				return &api.PoolWorkerDocument{
					PoolWorker: &api.PoolWorker{
						Buckets: []string{"one", "one", "one", "one", "one", "one", "two", "three"},
					},
				}
			},
			validate: func(t *testing.T, tt *test, doc *api.PoolWorkerDocument) {
				for i, bucket := range doc.PoolWorker.Buckets {
					if bucket != "" {
						t.Error(i, bucket)
					}
				}
			},
		},
		{
			name: "imbalanced",
			doc: func() *api.PoolWorkerDocument {
				return &api.PoolWorkerDocument{
					PoolWorker: &api.PoolWorker{
						Buckets: []string{"one", "one", "", "two", "one", "one", "one", "one"},
					},
				}
			},
			monitors: []string{"one", "two"},
			validate: func(t *testing.T, tt *test, doc *api.PoolWorkerDocument) {
				old := tt.doc()

				m := map[string]int{}
				for i, bucket := range doc.PoolWorker.Buckets {
					m[bucket]++
					switch bucket {
					case "one":
						if old.PoolWorker.Buckets[i] != bucket {
							t.Error(i)
						}
					case "two":
					default:
						t.Error(i, bucket)
					}
				}
				for k, v := range m {
					switch k {
					case "one", "two":
					default:
						t.Error(k)
					}
					if v != 4 {
						t.Error(k, v)
					}
				}
			},
		},
		{
			name: "stable",
			doc: func() *api.PoolWorkerDocument {
				return &api.PoolWorkerDocument{
					PoolWorker: &api.PoolWorker{
						Buckets: []string{"one", "two", "three", "one", "two", "three", "one", "three"},
					},
				}
			},
			monitors: []string{"one", "two", "three"},
			validate: func(t *testing.T, tt *test, doc *api.PoolWorkerDocument) {
				old := tt.doc()

				if !reflect.DeepEqual(old, doc) {
					t.Error(doc.PoolWorker.Buckets)
				}
			},
		},
		{
			name: "3->5",
			doc: func() *api.PoolWorkerDocument {
				return &api.PoolWorkerDocument{
					PoolWorker: &api.PoolWorker{
						Buckets: []string{"one", "two", "three", "one", "two", "three", "one", "three"},
					},
				}
			},
			monitors: []string{"one", "two", "three", "four", "five"},
			validate: func(t *testing.T, tt *test, doc *api.PoolWorkerDocument) {
				old := tt.doc()

				m := map[string]int{}
				for i, bucket := range doc.PoolWorker.Buckets {
					m[bucket]++
					switch bucket {
					case "one", "two", "three":
						if old.PoolWorker.Buckets[i] != bucket {
							t.Error(i)
						}
					case "four", "five":
					default:
						t.Error(i, bucket)
					}
				}
				for k, v := range m {
					switch k {
					case "one", "two", "three", "four", "five":
					default:
						t.Error(k)
					}
					if v > 2 {
						t.Error(k, v)
					}
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.doc()

			balance(tt.monitors, 8, doc)

			if doc.PoolWorker == nil {
				t.Fatal(doc.PoolWorker)
			}

			if len(doc.PoolWorker.Buckets) != 8 {
				t.Fatal(len(doc.PoolWorker.Buckets))
			}

			tt.validate(t, tt, doc)
		})
	}
}

func TestCapInterval(t *testing.T) {
	testCases := []struct {
		desc             string
		interval         time.Duration
		ttl              time.Duration
		expectedInterval time.Duration
		expectedTtl      time.Duration
		logs             []testlog.ExpectedLogEntry
	}{
		{
			desc:             "happy path",
			interval:         time.Second * 10,
			ttl:              time.Second * 60,
			expectedInterval: time.Second * 10,
			expectedTtl:      time.Second * 60,
			logs:             []testlog.ExpectedLogEntry{},
		},
		{
			desc:             "interval too short vs ttl",
			interval:         time.Second * 10,
			ttl:              time.Second * 10,
			expectedInterval: time.Millisecond * 7500,
			expectedTtl:      time.Second * 10,
			logs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("interval 10s was more than 75% of TTL 10s, capping"),
				},
			},
		},
		{
			desc:             "interval capped",
			interval:         time.Second * 50,
			ttl:              time.Second * 60,
			expectedInterval: time.Second * 45,
			expectedTtl:      time.Second * 60,
			logs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("interval must be at most 45s to align with renewLease, was 50s, capping"),
				},
			},
		},
		{
			desc:             "interval capped, then capped because lower TTL",
			interval:         time.Second * 100,
			ttl:              time.Second * 45,
			expectedInterval: time.Millisecond * (45000 * 0.75),
			expectedTtl:      time.Second * 45,
			logs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("interval must be at most 45s to align with renewLease, was 1m40s, capping"),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("interval 45s was more than 75% of TTL 45s, capping"),
				},
			},
		},
		{
			desc:             "ttl capped",
			interval:         time.Second * 10,
			ttl:              time.Second * 120,
			expectedInterval: time.Second * 10,
			expectedTtl:      time.Second * 60,
			logs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("workerTTL must be at most 1m0s to align with renewLease, was 2m0s, capping"),
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			r := require.New(t)
			hook, log := testlog.LogForTesting(t)
			gotInterval, gotTtl := capIntervals(log, tC.interval, tC.ttl)
			r.Equal(tC.expectedInterval, gotInterval)
			r.Equal(tC.expectedTtl, gotTtl)
			r.NoError(testlog.AssertLoggingOutput(hook, tC.logs))
		})
	}
}
