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
		name string
		v    *Version
		want Stream
		err  error
	}{
		{
			name: "upgrade when x.Y is lower than expected",
			v:    NewVersion(4, 3, 17),
			want: stream43,
		},
		{
			name: "no upgrade when x.Y is higher than exected",
			v:    NewVersion(4, 3, 19),
			err:  fmt.Errorf("not upgrading: cvo desired version is 4.3.19"),
		},
		{
			name: " when X.y id lower than exected",
			v:    NewVersion(4, 4, 2),
			want: stream44,
		},
		{
			name: "no upgrade when X.y is higher than expected",
			v:    NewVersion(4, 4, 9),
			err:  fmt.Errorf("not upgrading: cvo desired version is 4.4.9"),
		},
		{
			name: "cvo error",
			v:    NewVersion(4, 5, 1),
			err:  fmt.Errorf("not upgrading: stream not found 4.5.1"),
		},
		{
			name: "error",
			v:    NewVersion(5, 5, 1),
			err:  fmt.Errorf("not upgrading: stream not found 5.5.1"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			Streams = []Stream{stream43, stream44}
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
