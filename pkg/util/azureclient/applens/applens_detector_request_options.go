package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"time"

	gofrsuuid "github.com/gofrs/uuid"
)

// GetDetectorOptions includes options for GetDetector operation.
type GetDetectorOptions struct {
	ResourceID string
	DetectorID string
}

func (options *GetDetectorOptions) toHeader() http.Header {
	if options.ResourceID == "" || options.DetectorID == "" {
		return nil
	}

	id := gofrsuuid.Must(gofrsuuid.NewV4()).String()
	header := http.Header{
		headerXmsClientRequestId: {id},
		headerXmsDate:            {time.Now().UTC().Format(http.TimeFormat)},
		headerXmsPathQuery:       {fmt.Sprintf("%s/detectors/%s", options.ResourceID, options.DetectorID)},
		headerXmsRequestId:       {id},
		headerXmsVerb:            {"POST"},
	}

	return header
}

// ListDetectorOptions includes options for ListDetector operation.
type ListDetectorsOptions struct {
	ResourceID string
}

func (options *ListDetectorsOptions) toHeader() http.Header {
	if options.ResourceID == "" {
		return nil
	}

	id := gofrsuuid.Must(gofrsuuid.NewV4()).String()
	header := http.Header{
		headerXmsClientRequestId: {id},
		headerXmsDate:            {time.Now().UTC().Format(http.TimeFormat)},
		headerXmsPathQuery:       {fmt.Sprintf("%s/detectors", options.ResourceID)},
		headerXmsRequestId:       {id},
		headerXmsVerb:            {"POST"},
	}

	return header
}
