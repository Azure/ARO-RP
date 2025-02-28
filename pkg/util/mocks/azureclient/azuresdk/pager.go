package azuresdk

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/tracing"
)

// NewPager is a helper for creating a fake pager. The caller must still do the type gymnastics
// for wrapping the concrete type being paged in the List...Response{List...Result: { Value: []{items...} } }
// but there's no generic way to do that, so we can't provide a helper here for it.
func NewPager[T any](pages []T, errors []error) *runtime.Pager[T] {
	var currentPage int
	return runtime.NewPager[T](runtime.PagingHandler[T]{
		More: func(_ T) bool {
			return currentPage < len(pages)-1
		},
		Fetcher: func(ctx context.Context, t *T) (T, error) {
			page := pages[currentPage]
			err := errors[currentPage]
			currentPage++
			return page, err
		},
		Tracer: tracing.Tracer{},
	})
}
