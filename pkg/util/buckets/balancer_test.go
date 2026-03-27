package buckets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
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
