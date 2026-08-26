package changefeed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type fakeChangefeed struct {
	expectedPages int
	totalPages    int
	docs          []*api.OpenShiftClusterDocuments
	err           error
	stopChan      chan struct{}
}

func (f *fakeChangefeed) Next(ctx context.Context, limit int) (*api.OpenShiftClusterDocuments, error) {
	f.totalPages += 1
	if f.err != nil {
		y := f.err
		f.err = nil
		return nil, y
	}
	// close after a given number of pages have been processed
	if f.expectedPages == f.totalPages {
		close(f.stopChan)
	}
	if len(f.docs) == 0 {
		return nil, nil
	}
	x := f.docs[0]
	f.docs = f.docs[1:]
	return x, nil
}

type fakeResponder struct {
	docCount            int
	allPendingProcessed int
	locks               int
	unlocks             int
}

func (f *fakeResponder) OnAllPendingProcessed(gotAny bool) {
	f.allPendingProcessed += 1
}
func (f *fakeResponder) Lock()   { f.locks += 1 }
func (f *fakeResponder) Unlock() { f.unlocks += 1 }
func (f *fakeResponder) OnDoc(doc *api.OpenShiftClusterDocument) {
	f.docCount += 1
}

func TestChangefeedEmpty(t *testing.T) {
	h, log := testlog.LogForTesting(t)

	stopChan := make(chan struct{})

	r := &fakeResponder{}
	// not run in a goroutine because fakeChangefeed will deterministically
	// close stopchan and cause the changefeed loop to exit
	RunChangefeed(t.Context(), log, &noop.Noop{}, "OpenShiftClusterDocument", &fakeChangefeed{stopChan: stopChan, expectedPages: 1}, 100*time.Millisecond, 1, r, stopChan)

	err := testlog.AssertLoggingOutput(h, []testlog.ExpectedLogEntry{})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 0, r.docCount, "doc count")
	assert.Equal(t, 1, r.allPendingProcessed, "successful times all pending was processed")
	assert.Equal(t, 0, r.locks, "locks")
	assert.Equal(t, 0, r.unlocks, "unlocks")
}

func TestChangefeedSuccessfulDocs(t *testing.T) {
	h, log := testlog.LogForTesting(t)

	stopChan := make(chan struct{})

	cf := &fakeChangefeed{
		stopChan:      stopChan,
		expectedPages: 1,
		docs: []*api.OpenShiftClusterDocuments{{
			Count: 2, OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{{}, {}},
		}},
	}

	r := &fakeResponder{}
	// not run in a goroutine because fakeChangefeed will deterministically
	// close stopchan and cause the changefeed loop to exit
	RunChangefeed(t.Context(), log, &noop.Noop{}, "OpenShiftClusterDocument", cf, 100*time.Millisecond, 1, r, stopChan)

	err := testlog.AssertLoggingOutput(h, []testlog.ExpectedLogEntry{})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 2, r.docCount, "doc count")
	assert.Equal(t, 1, r.allPendingProcessed, "successful times all pending was processed")
	assert.Equal(t, 1, r.locks, "locks")
	assert.Equal(t, 1, r.unlocks, "unlocks")
}

func TestChangefeedProcessErrorContinuesProcessing(t *testing.T) {
	h, log := testlog.LogForTesting(t)

	stopChan := make(chan struct{})

	cf := &fakeChangefeed{
		expectedPages: 4,
		stopChan:      stopChan,
		err:           errors.New("test error"),
		docs: []*api.OpenShiftClusterDocuments{
			{
				Count: 2, OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{{}, {}},
			},
			nil,
			{
				Count: 1, OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{{}},
			},
		},
	}

	r := &fakeResponder{}
	// not run in a goroutine because fakeChangefeed will deterministically
	// close stopchan and cause the changefeed loop to exit
	RunChangefeed(t.Context(), log, &noop.Noop{}, "OpenShiftClusterDocument", cf, 1*time.Millisecond, 1, r, stopChan)

	err := testlog.AssertLoggingOutput(h, []testlog.ExpectedLogEntry{
		{
			"level": gomega.Equal(logrus.ErrorLevel),
			"msg":   gomega.Equal("while calling iterator.Next(): test error"),
		},
	})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 3, r.docCount, "doc count")
	assert.Equal(t, 2, r.allPendingProcessed, "successful times all pending was processed")
}

// recordingEmitter records every gauge emitted, in order.
type recordingEmitter struct {
	gauges     map[string][]int64
	dimensions map[string]map[string]string
}

func newRecordingEmitter() *recordingEmitter {
	return &recordingEmitter{
		gauges:     map[string][]int64{},
		dimensions: map[string]map[string]string{},
	}
}

func (e *recordingEmitter) EmitFloat(string, float64, map[string]string) {}

func (e *recordingEmitter) EmitGauge(name string, value int64, dimensions map[string]string) {
	e.gauges[name] = append(e.gauges[name], value)
	e.dimensions[name] = dimensions
}

// scriptedChangefeed returns errs in order, one per poll, and stops the
// changefeed once the script is exhausted.
type scriptedChangefeed struct {
	errs     []error
	polls    int
	stopChan chan struct{}
}

func (f *scriptedChangefeed) Next(ctx context.Context, limit int) (*api.OpenShiftClusterDocuments, error) {
	if f.polls >= len(f.errs) {
		return nil, nil
	}

	err := f.errs[f.polls]
	f.polls++
	if f.polls == len(f.errs) {
		close(f.stopChan)
	}

	return nil, err
}

// TestChangefeedEmitsFailureMetrics covers the half of this fix with value
// beyond the incident which prompted it. A stalled changefeed produced no
// signal at all for over a month; it must now report itself on every poll.
func TestChangefeedEmitsFailureMetrics(t *testing.T) {
	fail := errors.New("test error")

	for _, tt := range []struct {
		name         string
		errs         []error
		wantFailures []int64
	}{
		{
			name:         "a feed which cannot poll",
			errs:         []error{fail, fail, fail},
			wantFailures: []int64{1, 2, 3},
		},
		{
			name:         "a feed which recovers",
			errs:         []error{fail, fail, nil},
			wantFailures: []int64{1, 2, 0},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.LogForTesting(t)

			stopChan := make(chan struct{})
			m := newRecordingEmitter()

			// Not run in a goroutine: scriptedChangefeed deterministically
			// closes stopChan and causes the changefeed loop to exit.
			RunChangefeed(t.Context(), log, m, "OpenShiftClusterDocument",
				&scriptedChangefeed{errs: tt.errs, stopChan: stopChan},
				time.Millisecond, 1, &fakeResponder{}, stopChan)

			assert.Equal(t, tt.wantFailures, m.gauges[MetricConsecutiveFailures], MetricConsecutiveFailures)
			assert.Len(t, m.gauges[MetricStaleness], len(tt.errs), MetricStaleness+" is emitted on every poll")
			assert.Equal(t, map[string]string{"name": "OpenShiftClusterDocument"}, m.dimensions[MetricConsecutiveFailures], "dimensions")
		})
	}
}
