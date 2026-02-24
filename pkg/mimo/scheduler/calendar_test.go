package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"

	"github.com/go-test/deep"
)

func Test_parseCalendar(t *testing.T) {
	tests := []struct {
		s       string
		want    calendar
		wantErr string
	}{
		{
			s: "Mon *-*-* 00:00:00",
			want: calendar{
				Weekday: []time.Weekday{time.Monday},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{},

				Hour:   []int{0},
				Minute: []int{0},
			},
		},
		{
			s: "Mon *-*-* 00:00",
			want: calendar{
				Weekday: []time.Weekday{time.Monday},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{},

				Hour:   []int{0},
				Minute: []int{0},
			},
		},
		{
			s: "*-*-* 00:00",
			want: calendar{
				Weekday: []time.Weekday{},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{},

				Hour:   []int{0},
				Minute: []int{0},
			},
		},
		{
			s: "*-*-1 00:00:00",
			want: calendar{
				Weekday: []time.Weekday{},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{1},

				Hour:   []int{0},
				Minute: []int{0},
			},
		},
		{
			s: "*-*-4,1,2,3 00:00:00",
			want: calendar{
				Weekday: []time.Weekday{},
				Year:    []int{},
				Month:   []int{},
				// output is sorted
				Day: []int{1, 2, 3, 4},

				Hour:   []int{0},
				Minute: []int{0},
			},
		},
		{
			s: "2000-*-* 00:00:00",
			want: calendar{
				Weekday: []time.Weekday{},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{},
				Hour:    []int{},
				Minute:  []int{},
			},
			wantErr: "error parsing year: 2000 is out of range (before 2026)",
		},
		{
			s: "2026-01-01 00:00:*",
			want: calendar{
				Weekday: []time.Weekday{},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{},
				Hour:    []int{},
				Minute:  []int{},
			},
			wantErr: "error parsing seconds: per-second granularity is unsupported",
		},
		{
			s: "garbage",
			want: calendar{
				Weekday: []time.Weekday{},
				Year:    []int{},
				Month:   []int{},
				Day:     []int{},
				Hour:    []int{},
				Minute:  []int{},
			},
			wantErr: "failed to match 'garbage'",
		},
	}
	for _, tt := range tests {
		t.Run("parsing "+tt.s, func(t *testing.T) {
			got, gotErr := ParseCalendar(tt.s)
			if gotErr != nil {
				for _, e := range deep.Equal(gotErr.Error(), tt.wantErr) {
					t.Error(e)
				}
				return
			}
			if tt.wantErr != "" {
				t.Fatal("parseCalendar() succeeded unexpectedly")
			}
			for _, e := range deep.Equal(got, tt.want) {
				t.Error(e)
			}
		})
	}
}

func TestNext(t *testing.T) {
	tests := []struct {
		name  string // description of this test case
		now   time.Time
		sched calendar
		want  string
		want2 bool
	}{
		{
			name:  "fixed year",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Year: []int{2028}, Month: []int{1}, Day: []int{1}, Hour: []int{0}, Minute: []int{0}},
			want:  "2028-01-01 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next month, over year",
			now:   time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Day: []int{1}, Hour: []int{0}, Minute: []int{0}},
			want:  "2027-01-01 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next day",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Hour: []int{0}, Minute: []int{0}},
			want:  "2026-01-02 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next monday",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Weekday: []time.Weekday{time.Monday}, Hour: []int{0}, Minute: []int{0}},
			want:  "2026-01-05 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next monday when it is tomorrow",
			now:   time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
			sched: calendar{Weekday: []time.Weekday{time.Monday}, Hour: []int{0}, Minute: []int{0}},
			want:  "2026-01-05 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next monday when today is a monday",
			now:   time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			sched: calendar{Weekday: []time.Weekday{time.Monday}, Hour: []int{0}, Minute: []int{0}},
			want:  "2026-01-12 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next day (feb 28+1 26)",
			now:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
			sched: calendar{Hour: []int{0}, Minute: []int{0}},
			want:  "2026-03-01 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next day (feb 28+1 2028, leap year)",
			now:   time.Date(2028, 2, 28, 0, 0, 0, 0, time.UTC),
			sched: calendar{Hour: []int{0}, Minute: []int{0}},
			want:  "2028-02-29 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next hour",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Minute: []int{0}},
			want:  "2026-01-01 01:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next hour, over day",
			now:   time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC),
			sched: calendar{Minute: []int{0}},
			want:  "2026-01-02 00:00:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next half hour",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Minute: []int{30}},
			want:  "2026-01-01 00:30:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next 15 mins",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Minute: []int{0, 15, 30, 45}},
			want:  "2026-01-01 00:15:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "next 15 mins @ 15 past",
			now:   time.Date(2026, 1, 1, 0, 15, 0, 0, time.UTC),
			sched: calendar{Minute: []int{0, 15, 30, 45}},
			want:  "2026-01-01 00:30:00 +0000 UTC",
			want2: true,
		},
		{
			name:  "never",
			now:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			sched: calendar{Year: []int{2025}},
			want2: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := Next(tt.now, tt.sched)
			if got2 {
				for _, e := range deep.Equal(got.String(), tt.want) {
					t.Error(e)
				}
			}
			for _, e := range deep.Equal(got2, tt.want2) {
				t.Error(e)
			}
		})
	}
}
