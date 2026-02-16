package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Generic list of documents that ___DocumentList implements, for use in
// changefeeds
type DocumentList[E any] interface {
	Docs() []E
	GetCount() int
}
