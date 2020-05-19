package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOpenShiftVersions(t *testing.T) {
	for _, u := range Streams {
		_, err := ParseVersion(u.Version.String())
		if err != nil {
			t.Error(err)
		}
	}
}

func TestUnique(t *testing.T) {
	unique := make(map[string]int, len(Streams))
	for _, u := range Streams {
		unique[fmt.Sprintf("%d.%d", u.Version.V[0], u.Version.V[1])]++
	}

	for i, j := range unique {
		if j > 1 {
			t.Errorf("multiple x.Y version upgrade path found for %s", i)
		}
	}
}

func TestGetStream(t *testing.T) {
	Stream43 := Stream{
		Version: NewVersion(4, 3, 18),
	}
	Stream44 := Stream{
		Version: NewVersion(4, 4, 3),
	}

	Streams = append([]Stream{}, Stream43, Stream44)
	for _, tt := range []struct {
		name string
		v    *Version
		want Stream
		err  error
	}{
		{
			name: "4.3 - upgrade",
			v:    NewVersion(4, 3, 17),
			want: Stream43,
		},
		{
			name: "4.3 - error",
			v:    NewVersion(4, 3, 19),
			err:  fmt.Errorf("not upgrading: cvo desired version is 4.3.19"),
		},
		{
			name: "4.4 - upgrade",
			v:    NewVersion(4, 4, 2),
			want: Stream44,
		},
		{
			name: "4.4.10 stream",
			v:    NewVersion(4, 4, 9),
			err:  fmt.Errorf("not upgrading: cvo desired version is 4.4.9"),
		},
		{
			name: "error",
			v:    NewVersion(4, 5, 1),
			err:  fmt.Errorf("stream for 4.5.1 not found"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStream(tt.v)
			if err != nil && tt.err != nil && !reflect.DeepEqual(tt.err, err) {
				t.Fatal(err)
			}
			if got != nil && !reflect.DeepEqual(got, &tt.want) {
				t.Error(cmp.Diff(got, &tt.want))
			}
		})
	}
}
