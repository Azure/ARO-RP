package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"github.com/go-test/deep"
)

func TestMonitorEmit(t *testing.T) {

	type test struct {
		name               string
		startTime          time.Time
		startGracePeriod   time.Duration
		lastBucketTime     time.Time
		lastChangefeedTime time.Time
		expectFailed       bool
		expectErrors       map[string]string
	}

	for _, tt := range []*test{
		{
			name:             "Failures allowed in grace period, haven't got bucket or changefeed time",
			startTime:        time.Unix(900, 0),
			startGracePeriod: time.Minute * 2,
			expectFailed:     false,
			expectErrors: map[string]string{
				"lastBucketTime":     "buckets not yet read",
				"lastChangefeedTime": "changefeed not yet read",
			},
		},
		{
			name:               "Failures allowed in grace period",
			startTime:          time.Unix(900, 0),
			startGracePeriod:   time.Minute * 2,
			lastBucketTime:     time.Unix(910, 0),
			lastChangefeedTime: time.Unix(910, 0),
			expectFailed:       false,
			expectErrors: map[string]string{
				"lastBucketTime":     "running behind, 1m30s > 1m0s",
				"lastChangefeedTime": "running behind, 1m30s > 1m0s",
			},
		},
		{
			name:             "Failures not allowed outside grace period, haven't got bucket or changefeed time",
			startTime:        time.Unix(900, 0),
			startGracePeriod: time.Second * 1,
			expectFailed:     true,
			expectErrors: map[string]string{
				"lastBucketTime":     "buckets not yet read",
				"lastChangefeedTime": "changefeed not yet read",
			},
		},
		{
			name:               "Failures not allowed outside grace period",
			startTime:          time.Unix(900, 0),
			startGracePeriod:   time.Second * 1,
			lastBucketTime:     time.Unix(910, 0),
			lastChangefeedTime: time.Unix(910, 0),
			expectFailed:       true,
			expectErrors: map[string]string{
				"lastBucketTime":     "running behind, 1m30s > 1m0s",
				"lastChangefeedTime": "running behind, 1m30s > 1m0s",
			},
		},
		{
			name:               "Success",
			startTime:          time.Unix(900, 0),
			startGracePeriod:   time.Second * 1,
			lastBucketTime:     time.Unix(999, 0),
			lastChangefeedTime: time.Unix(999, 0),
			expectFailed:       false,
			expectErrors:       map[string]string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			now := func() time.Time { return time.Unix(1000, 0) }
			mon := &monitor{
				now:              now,
				startTime:        tt.startTime,
				startGracePeriod: tt.startGracePeriod,
			}

			if !tt.lastBucketTime.IsZero() {
				mon.lastBucketlist.Store(tt.lastBucketTime)
			}
			if !tt.lastChangefeedTime.IsZero() {
				mon.lastChangefeed.Store(tt.lastChangefeedTime)
			}

			failed, failing := mon.checkReady()

			if failed != tt.expectFailed {
				t.Errorf("supposed to report %t, got %t", tt.expectFailed, failed)
			}

			diffs := deep.Equal(tt.expectErrors, failing)
			for _, x := range diffs {
				t.Error(x)
			}
		})
	}
}
