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

func TestGetUpgradeStream(t *testing.T) {
	stream43 := Stream{
		Version: NewVersion(4, 3, 18),
	}
	stream44 := Stream{
		Version: NewVersion(4, 4, 3),
	}

	for _, tt := range []struct {
		name    string
		v       *Version
		streams []Stream
		want    Stream
		err     error
	}{
		{
			name:    "upgrade x.Y lower",
			v:       NewVersion(4, 3, 17),
			streams: []Stream{stream43, stream44},
			want:    stream43,
		},
		{
			name:    "upgrade x.Y higher",
			v:       NewVersion(4, 3, 19),
			streams: []Stream{stream43, stream44},
			want:    stream43,
		},
		{
			name:    "upgrade X higher",
			v:       NewVersion(4, 4, 2),
			streams: []Stream{stream43, stream44},
			want:    stream44,
		},
		{
			name:    "upgrade X lower",
			v:       NewVersion(4, 4, 9),
			streams: []Stream{stream43, stream44},
			want:    stream44,
		},
		{
			name:    "cvo error",
			v:       NewVersion(4, 5, 1),
			streams: []Stream{stream43, stream44},
			err:     fmt.Errorf("not upgrading: stream not found 4.5.1"),
		},
		{
			name:    "error",
			v:       NewVersion(5, 5, 1),
			streams: []Stream{stream43, stream44},
			err:     fmt.Errorf("not upgrading: stream not found 5.5.1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			Streams = tt.streams
			got, err := GetUpgradeStream(tt.v)
			if err != nil && tt.err != nil && !reflect.DeepEqual(tt.err, err) {
				t.Fatal(err)
			}
			if got != nil && !reflect.DeepEqual(got, &tt.want) {
				t.Error(cmp.Diff(got, &tt.want))
			}
		})
	}
}
