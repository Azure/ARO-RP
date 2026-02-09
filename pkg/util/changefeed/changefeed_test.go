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

func (f *fakeResponder) OnAllPendingProcessed() {
	f.allPendingProcessed += 1
}
func (f *fakeResponder) Lock()   { f.locks += 1 }
func (f *fakeResponder) Unlock() { f.unlocks += 1 }
func (f *fakeResponder) OnDoc(doc *api.OpenShiftClusterDocument) {
	f.docCount += 1
}

func TestChangefeedEmpty(t *testing.T) {
	h, log := testlog.New()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stopChan := make(chan struct{})

	r := &fakeResponder{}
	NewChangefeed(ctx, log, &fakeChangefeed{stopChan: stopChan, expectedPages: 1}, 100*time.Millisecond, 1, r, stopChan)

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
	h, log := testlog.New()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stopChan := make(chan struct{})

	cf := &fakeChangefeed{
		stopChan:      stopChan,
		expectedPages: 1,
		docs: []*api.OpenShiftClusterDocuments{{
			Count: 1, OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{{}, {}},
		}},
	}

	r := &fakeResponder{}
	NewChangefeed(ctx, log, cf, 100*time.Millisecond, 1, r, stopChan)

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
	h, log := testlog.New()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

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
	NewChangefeed(ctx, log, cf, 1*time.Millisecond, 1, r, stopChan)

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
