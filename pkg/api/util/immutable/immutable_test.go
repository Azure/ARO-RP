package immutable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
	"time"
)

type ts struct {
	Mutable     string            `json:"mutable,omitempty" mutable:"true"` // should be able to change
	Case        string            `json:"case,omitempty" mutable:"case"`    // should be case insensitive
	Empty       string            `json:"empty,omitempty" mutable:""`       // default to immutable
	Map         map[string]string `json:"map,omitempty"`
	EmptyNoJSON string            `mutable:"false"` // handle no json tag
	None        string            // default to immutable
	Time        time.Time         `json:"time,omitempty"`
	MutableTime time.Time         `json:"mutableTime,omitempty" mutable:"true"`
}

func TestValidate(t *testing.T) {
	before := ts{
		Mutable:     "before",
		Case:        "before",
		Empty:       "before",
		EmptyNoJSON: "before",
		None:        "before",
		Map: map[string]string{
			"key": "value",
		},
	}
	tests := []struct {
		name    string
		modify  func(*ts)
		wantErr string
	}{
		{
			name: "no change",
		},
		{
			name: "can change mutable",
			modify: func(s *ts) {
				s.Mutable = "after"
			},
		},
		{
			name: "can change case caps",
			modify: func(s *ts) {
				s.Case = "BEFORE"
			},
		},
		{
			name: "can NOT change case",
			modify: func(s *ts) {
				s.Case = "after"
			},
			wantErr: "Changing property 'case' is not allowed.",
		},
		{
			name: "can NOT change empty",
			modify: func(s *ts) {
				s.Empty = "after"
			},
			wantErr: "Changing property 'empty' is not allowed.",
		},
		{
			name: "can NOT replace a map",
			modify: func(s *ts) {
				s.Map = map[string]string{"new": "value"}
			},
			wantErr: "Changing property 'map' is not allowed.",
		},
		{
			name: "can NOT change a value in a map",
			modify: func(s *ts) {
				s.Map = map[string]string{"key": "new-value"}
			},
			wantErr: `Changing property 'map["key"]' is not allowed.`,
		},
		{
			name: "can NOT change EmptyNoJSON",
			modify: func(s *ts) {
				s.EmptyNoJSON = "after"
			},
			wantErr: "Changing property 'EmptyNoJSON' is not allowed.",
		},
		{
			name: "can NOT change None",
			modify: func(s *ts) {
				s.None = "after"
			},
			wantErr: "Changing property 'None' is not allowed.",
		},
		{
			name: "can NOT change Time",
			modify: func(s *ts) {
				s.Time = time.Unix(0, 0)
			},
			wantErr: "Changing property 'time.ext' is not allowed.",
		},
		{
			name: "can change MutableTime",
			modify: func(s *ts) {
				s.MutableTime = time.Unix(0, 0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := before

			if tt.modify != nil {
				tt.modify(&after)
			}

			err := Validate("", &after, &before)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}
			} else {
				if err.Error() != tt.wantErr {
					t.Error(err)
				}

				_, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("%T", err)
				}
			}
		})
	}
}
