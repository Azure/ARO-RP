package api

import "testing"

type ts struct {
	Case        string `json:"case,omitempty" mutable:"case"`    // should be case insensitive
	Mutable     string `json:"mutable,omitempty" mutable:"true"` // should be able to change
	Empty       string `json:"empty,omitempty" mutable:""`       // default to immutable
	EmptyNoJSON string `mutable:"false"`                         // handle no json tag
	None        string // default to immutable
}

func TestValidateImmutable(t *testing.T) {
	before := ts{
		Mutable:     "before",
		Case:        "before",
		Empty:       "before",
		EmptyNoJSON: "before",
		None:        "before",
	}
	tests := []struct {
		name    string
		modify  func(s *ts)
		wantErr string
	}{
		{
			name:   "no change",
			modify: func(s *ts) {},
		},
		{
			name: "can change mutables",
			modify: func(s *ts) {
				s.Mutable = "what ever I want"
			},
		},
		{
			name: "can change Case caps",
			modify: func(s *ts) {
				s.Case = "BeFoRe"
			},
		},
		{
			name: "can NOT change None",
			modify: func(s *ts) {
				s.None = "what ever i want"
			},
			wantErr: "400: PropertyChangeNotAllowed: None: Changing property 'None' is not allowed.",
		},
		{
			name: "can NOT change Empty",
			modify: func(s *ts) {
				s.Empty = "what ever i want"
			},
			wantErr: "400: PropertyChangeNotAllowed: empty: Changing property 'empty' is not allowed.",
		},
		{
			name: "can NOT change EmptyNoJSON",
			modify: func(s *ts) {
				s.EmptyNoJSON = "what ever i want"
			},
			wantErr: "400: PropertyChangeNotAllowed: EmptyNoJSON: Changing property 'EmptyNoJSON' is not allowed.",
		},
		{
			name: "can NOT change Case",
			modify: func(s *ts) {
				s.Case = "what ever i want"
			},
			wantErr: "400: PropertyChangeNotAllowed: case: Changing property 'case' is not allowed.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := before
			tt.modify(&after)
			err := ValidateImmutable("", &after, &before)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}
			} else {
				if err.Error() != tt.wantErr {
					t.Error(err)
				}
			}

		})
	}
}
