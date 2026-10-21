package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"testing"
)

type fakeRebuilder struct {
	calls int
	err   error
}

func (r *fakeRebuilder) Rebuild() error {
	r.calls++
	return r.err
}

func TestMultiRebuilder(t *testing.T) {
	wantErr := errors.New("boom")

	for _, tt := range []struct {
		name      string
		first     *fakeRebuilder
		second    *fakeRebuilder
		wantErr   error
		wantCalls []int
	}{
		{
			name:      "rebuilds all",
			first:     &fakeRebuilder{},
			second:    &fakeRebuilder{},
			wantCalls: []int{1, 1},
		},
		{
			name:      "stops at first error",
			first:     &fakeRebuilder{err: wantErr},
			second:    &fakeRebuilder{},
			wantErr:   wantErr,
			wantCalls: []int{1, 0},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMultiRebuilder(tt.first, nil, tt.second).Rebuild()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, wanted %v", err, tt.wantErr)
			}
			if tt.first.calls != tt.wantCalls[0] || tt.second.calls != tt.wantCalls[1] {
				t.Errorf("got calls %d,%d wanted %v", tt.first.calls, tt.second.calls, tt.wantCalls)
			}
		})
	}
}
