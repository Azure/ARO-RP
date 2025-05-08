package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.s

import (
	"testing"
)

func TestGetDetectorOptionsToHeader(t *testing.T) {
	options := &GetDetectorOptions{}
	if options.toHeader() != nil {
		t.Error("toHeader should return nil")
	}

	options.ResourceID = ""
	options.DetectorID = ""
	header := options.toHeader()
	if header != nil {
		t.Fatal("toHeader should return nil")
	}

	options.DetectorID = "testdetector"
	header = options.toHeader()
	if header != nil {
		t.Fatal("toHeader should return nil")
	}

	options.ResourceID = "testresourceid"
	header = options.toHeader()
	if header == nil {
		t.Fatal("toHeader should return non-nil")
	}

	options.DetectorID = ""
	header = options.toHeader()
	if header != nil {
		t.Fatal("toHeader should return nil")
	}
}

func TestListDetectorsOptionsToHeader(t *testing.T) {
	options := &ListDetectorsOptions{}
	if options.toHeader() != nil {
		t.Error("toHeader should return nil")
	}

	options.ResourceID = ""
	header := options.toHeader()
	if header != nil {
		t.Fatal("toHeader should return nil")
	}

	options.ResourceID = "testresourceid"
	header = options.toHeader()
	if header == nil {
		t.Fatal("toHeader should return non-nil")
	}
}
