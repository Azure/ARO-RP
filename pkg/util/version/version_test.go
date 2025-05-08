package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNewVersion(t *testing.T) {
	for i, tt := range []struct {
		vs   []uint32
		want *Version
	}{
		{
			vs:   []uint32{1, 2},
			want: &Version{V: [3]uint32{1, 2}},
		},
		{
			want: &Version{},
		},
		{
			vs:   []uint32{1, 2, 3, 4},
			want: &Version{V: [3]uint32{1, 2, 3}},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := NewVersion(tt.vs...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	for _, tt := range []struct {
		vsn     string
		want    *Version
		wantErr string
	}{
		{
			vsn:  "4.3.0-0.nightly-2020-04-17-062811",
			want: &Version{V: [3]uint32{4, 3}, Suffix: "-0.nightly-2020-04-17-062811"},
		},
		{
			vsn:  "40.30.10",
			want: &Version{V: [3]uint32{40, 30, 10}},
		},
		{
			vsn:  " 40.30.10 ",
			want: &Version{V: [3]uint32{40, 30, 10}},
		},
		{
			vsn:  "4000.3000.1000",
			want: &Version{V: [3]uint32{4000, 3000, 1000}},
		},
		{
			vsn:     "bad",
			wantErr: `could not parse version "bad"`,
		},
	} {
		t.Run(tt.vsn, func(t *testing.T) {
			got, err := ParseVersion(tt.vsn)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestLt(t *testing.T) {
	for i, tt := range []struct {
		input *Version
		min   *Version
		want  bool
	}{
		{
			input: NewVersion(4, 1),
			min:   NewVersion(4, 3),
			want:  true,
		},
		{
			input: NewVersion(4, 4),
			min:   NewVersion(4, 3, 1),
		},
		{
			input: NewVersion(4, 4),
			min:   NewVersion(4, 4),
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := tt.input.Lt(tt.min)
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}

func TestEq(t *testing.T) {
	for i, tt := range []struct {
		input *Version
		vsn   string
		equal bool
	}{
		{
			input: NewVersion(4, 4, 10),
			vsn:   "4.4.10",
			equal: true,
		},
		{
			input: NewVersion(4, 1, 10),
			vsn:   "4.3.10",
		},
		{
			input: NewVersion(4, 4),
			vsn:   "4.3.1",
		},
		{
			input: NewVersion(4, 4, 10),
			vsn:   "4.4.10-rc1",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			vsn, err := ParseVersion(tt.vsn)
			if err != nil {
				t.Error(err)
			}

			got := tt.input.Eq(vsn)
			if got != tt.equal {
				t.Error(got)
			}
		})
	}
}
