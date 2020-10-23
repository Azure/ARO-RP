package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"
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
	stream45 := Stream{
		Version: NewVersion(4, 5, 0),
	}

	for _, tt := range []struct {
		name     string
		v        *Version
		want     *Stream
		upgradeY bool
		streams  []*Stream
	}{
		{
			name:     "upgrade when Y versions match and candidate Z (4.3.18) is greater",
			v:        NewVersion(4, 3, 17),
			upgradeY: false,
			streams:  []*Stream{&stream43, &stream44},
			want:     &stream43,
		},
		{
			name:     "don't upgrade when Y versions match but current Z (4.3.19) is greater",
			v:        NewVersion(4, 3, 19),
			upgradeY: false,
			streams:  []*Stream{&stream43, &stream44},
			want:     nil,
		},
		{
			name:     "upgrade when Y versions match and candidate Z (4.4.3) is greater",
			v:        NewVersion(4, 4, 2),
			upgradeY: false,
			streams:  []*Stream{&stream43, &stream44},
			want:     &stream44,
		},
		{
			name:     "don't upgrade when Y versions match but current Z (4.4.9) is greater",
			v:        NewVersion(4, 4, 9),
			upgradeY: false,
			streams:  []*Stream{&stream43, &stream44},
			want:     nil,
		},
		{
			name:     "upgrade to Y+1 when allowed and candidate y.Z (4.3.18) < current y.Z (4.3.19)",
			v:        NewVersion(4, 3, 19),
			upgradeY: true,
			streams:  []*Stream{&stream43, &stream44},
			want:     &stream44,
		},
		{
			name:     "upgrade to Y+1 when allowed and candidate y.Z == current y.Z (4.3.18)",
			v:        stream43.Version,
			upgradeY: true,
			streams:  []*Stream{&stream43, &stream44},
			want:     &stream44,
		},
		{
			name:     "upgrade to Y+1 (not Y+2) when allowed and candidate y.Z == current y.Z (4.3.18)",
			v:        stream43.Version,
			upgradeY: true,
			streams:  []*Stream{&stream43, &stream44, &stream45},
			want:     &stream44,
		},
		{
			name:     "don't upgrade Y when not allowed",
			v:        stream43.Version,
			upgradeY: false,
			streams:  []*Stream{&stream43, &stream44},
			want:     nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			Streams = tt.streams
			got := GetUpgradeStream(tt.v, tt.upgradeY)
			if got != nil && tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wanted %s, got %s", tt.want.Version, got.Version)
			}
			if got == nil && tt.want != nil {
				t.Errorf("wanted %s, got nil", tt.want.Version)
			}
			if got != nil && tt.want == nil {
				t.Errorf("wanted nil, got %s", got)
			}
		})
	}
}
