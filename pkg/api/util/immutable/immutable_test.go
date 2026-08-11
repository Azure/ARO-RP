package immutable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
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

type policyChild struct {
	Name     *string
	ClientID *string
}

type policyModel struct {
	ID              *string
	Location        *string
	Mutable         *string
	ReadOnly        *string
	StrictReadOnly  *string
	MutableReadOnly *string
	ReadOnlyText    string
	Child           *policyChild
	OptionalChild   *policyChild
	Items           []*policyChild
	Labels          map[string]string
	Numbers         []string
}

func TestValidateWithPolicy(t *testing.T) {
	beforeValue := "before"
	afterValue := "after"
	name := "worker"
	before := &policyModel{
		ID:       &beforeValue,
		Location: &beforeValue,
		Mutable:  &beforeValue,
		Child:    &policyChild{},
		Items:    []*policyChild{{Name: &name, ClientID: &beforeValue}},
	}
	policy := Policy{
		Mutable:         []string{"mutable", "mutableReadOnly"},
		ReadOnly:        []string{"readOnly", "strictReadOnly", "mutableReadOnly", "readOnlyText"},
		ReadOnlyValue:   []string{"readOnly"},
		CaseInsensitive: []string{"id"},
		NormalizeNil:    []string{"child"},
	}

	tests := []struct {
		name    string
		modify  func(*policyModel)
		wantErr string
	}{
		{name: "no change"},
		{
			name: "mutable change",
			modify: func(model *policyModel) {
				model.Mutable = &afterValue
			},
		},
		{
			name: "case-insensitive change",
			modify: func(model *policyModel) {
				value := "BEFORE"
				model.ID = &value
			},
		},
		{
			name: "case-insensitive field rejects different value",
			modify: func(model *policyModel) {
				model.ID = &afterValue
			},
			wantErr: "Changing property 'id' is not allowed.",
		},
		{
			name: "immutable pointer change",
			modify: func(model *policyModel) {
				model.Location = &afterValue
			},
			wantErr: "Changing property 'location' is not allowed.",
		},
		{
			name: "read-only request value",
			modify: func(model *policyModel) {
				model.ReadOnly = &afterValue
			},
			wantErr: "Changing property 'readOnly' is not allowed.",
		},
		{
			name: "read-only pointed zero value",
			modify: func(model *policyModel) {
				value := ""
				model.ReadOnly = &value
			},
		},
		{
			name: "strict read-only pointed zero value",
			modify: func(model *policyModel) {
				value := ""
				model.StrictReadOnly = &value
			},
			wantErr: "Changing property 'strictReadOnly' is not allowed.",
		},
		{
			name: "mutable takes precedence over read-only",
			modify: func(model *policyModel) {
				model.MutableReadOnly = &afterValue
			},
		},
		{
			name: "non-pointer read-only value",
			modify: func(model *policyModel) {
				model.ReadOnlyText = "after"
			},
			wantErr: "Changing property 'readOnlyText' is not allowed.",
		},
		{
			name: "named slice path",
			modify: func(model *policyModel) {
				model.Items[0].ClientID = &afterValue
			},
			wantErr: "Changing property 'items['worker'].clientId' is not allowed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := *before
			after.Items = []*policyChild{{Name: before.Items[0].Name, ClientID: before.Items[0].ClientID}}
			if tt.modify != nil {
				tt.modify(&after)
			}

			err := ValidateWithPolicy("", &after, before, policy)
			if err == nil && tt.wantErr != "" {
				t.Fatalf("expected %q", tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantErr {
				t.Fatalf("got %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWithPolicyPointers(t *testing.T) {
	beforeValue := "before"
	afterValue := "after"
	before := &policyModel{
		Location:      nil,
		Child:         &policyChild{},
		OptionalChild: &policyChild{},
	}
	policy := Policy{NormalizeNil: []string{"child"}}

	tests := []struct {
		name    string
		modify  func(*policyModel)
		wantErr string
	}{
		{
			name: "nil scalar equals pointed zero",
			modify: func(model *policyModel) {
				value := ""
				model.Location = &value
			},
		},
		{
			name: "nil scalar differs from pointed value",
			modify: func(model *policyModel) {
				model.Location = &afterValue
			},
			wantErr: "Changing property 'location' is not allowed.",
		},
		{
			name: "normalized nil struct equals empty struct",
			modify: func(model *policyModel) {
				model.Child = nil
			},
		},
		{
			name: "normalized nil struct differs from nonzero struct",
			modify: func(model *policyModel) {
				model.Child = &policyChild{ClientID: &beforeValue}
			},
			wantErr: "Changing property 'child.clientId' is not allowed.",
		},
		{
			name: "non-normalized nil struct differs from empty struct",
			modify: func(model *policyModel) {
				model.OptionalChild = nil
			},
			wantErr: "Changing property 'optionalChild' is not allowed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := *before
			if tt.modify != nil {
				tt.modify(&after)
			}

			assertPolicyValidation(t, &after, before, policy, tt.wantErr)
		})
	}
}

func TestValidateWithPolicyCollections(t *testing.T) {
	beforeValue := "before"
	afterValue := "after"
	name := "worker"
	before := &policyModel{
		Items:   []*policyChild{{Name: &name, ClientID: &beforeValue}},
		Labels:  map[string]string{"key": "value"},
		Numbers: []string{"one"},
	}

	tests := []struct {
		name    string
		modify  func(*policyModel)
		policy  Policy
		wantErr string
	}{
		{
			name: "named slice element path",
			modify: func(model *policyModel) {
				model.Items[0].ClientID = &afterValue
			},
			wantErr: "Changing property 'items['worker'].clientId' is not allowed.",
		},
		{
			name: "unnamed slice element uses numeric path",
			modify: func(model *policyModel) {
				model.Items[0].Name = nil
				model.Items[0].ClientID = &afterValue
			},
			wantErr: "Changing property 'items[0].name' is not allowed.",
		},
		{
			name: "slice length change",
			modify: func(model *policyModel) {
				model.Items = append(model.Items, &policyChild{})
			},
			wantErr: "Changing property 'items' is not allowed.",
		},
		{
			name: "nil slice differs from empty slice",
			modify: func(model *policyModel) {
				model.Numbers = nil
			},
			wantErr: "Changing property 'numbers' is not allowed.",
		},
		{
			name: "map value path",
			modify: func(model *policyModel) {
				model.Labels["key"] = "changed"
			},
			wantErr: `Changing property 'labels["key"]' is not allowed.`,
		},
		{
			name: "map key change",
			modify: func(model *policyModel) {
				model.Labels = map[string]string{"other": "value"}
			},
			wantErr: "Changing property 'labels' is not allowed.",
		},
		{
			name: "mutable collection skips descendants",
			modify: func(model *policyModel) {
				model.Items[0].ClientID = &afterValue
			},
			policy: Policy{Mutable: []string{"items"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := *before
			after.Items = []*policyChild{{Name: before.Items[0].Name, ClientID: before.Items[0].ClientID}}
			after.Labels = map[string]string{"key": "value"}
			after.Numbers = append([]string(nil), before.Numbers...)
			tt.modify(&after)

			assertPolicyValidation(t, &after, before, tt.policy, tt.wantErr)
		})
	}

	emptyElement := &policyModel{Items: []*policyChild{{}}}
	nilElement := &policyModel{Items: []*policyChild{nil}}
	assertPolicyValidation(t, nilElement, emptyElement, Policy{}, "")
}

func TestValidateWithPolicyEmbeddedField(t *testing.T) {
	type Embedded struct {
		ClientID *string
	}
	type model struct {
		Embedded
	}

	beforeValue := "before"
	afterValue := "after"
	before := &model{Embedded: Embedded{ClientID: &beforeValue}}
	after := &model{Embedded: Embedded{ClientID: &afterValue}}

	assertPolicyValidation(t, after, before, Policy{}, "Changing property 'clientId' is not allowed.")
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		value    string
		want     bool
	}{
		{name: "exact match", patterns: []string{"properties.tags"}, value: "properties.tags", want: true},
		{name: "exact mismatch", patterns: []string{"properties.tags"}, value: "properties.name"},
		{name: "middle wildcard", patterns: []string{"properties.ingressProfiles*.ip"}, value: "properties.ingressProfiles['default'].ip", want: true},
		{name: "wildcard matches empty", patterns: []string{"properties.*tags"}, value: "properties.tags", want: true},
		{name: "multiple wildcards", patterns: []string{"properties.*Profiles*.ip"}, value: "properties.ingressProfiles['default'].ip", want: true},
		{name: "prefix must match", patterns: []string{"properties.*.ip"}, value: "other.profile.ip"},
		{name: "suffix must match", patterns: []string{"properties.*.ip"}, value: "properties.profile.url"},
		{name: "intermediate part must match", patterns: []string{"properties.*Profiles*.ip"}, value: "properties.network['default'].ip"},
		{name: "empty patterns", value: "properties.tags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesAny(tt.patterns, tt.value); got != tt.want {
				t.Fatalf("matchesAny(%q, %q) = %t, want %t", tt.patterns, tt.value, got, tt.want)
			}
		})
	}
}

func TestPolicyJSONName(t *testing.T) {
	type model struct {
		Tagged               string `json:"renamed,omitempty"`
		ID                   string
		ClientID             string
		EffectiveOutboundIPs string
		URL                  string
		OidcIssuer           string
		VMSize               string
		Location             string
	}

	typeOfModel := reflect.TypeOf(model{})
	tests := []struct {
		field string
		want  string
	}{
		{field: "Tagged", want: "renamed"},
		{field: "ID", want: "id"},
		{field: "ClientID", want: "clientId"},
		{field: "EffectiveOutboundIPs", want: "effectiveOutboundIps"},
		{field: "URL", want: "url"},
		{field: "OidcIssuer", want: "oidcIssuer"},
		{field: "VMSize", want: "vmSize"},
		{field: "Location", want: "location"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			field, ok := typeOfModel.FieldByName(tt.field)
			if !ok {
				t.Fatalf("field %q not found", tt.field)
			}
			if got := policyJSONName(field); got != tt.want {
				t.Fatalf("policyJSONName(%s) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

func assertPolicyValidation(t *testing.T, after, before interface{}, policy Policy, wantErr string) {
	t.Helper()

	err := ValidateWithPolicy("", after, before, policy)
	if err == nil && wantErr != "" {
		t.Fatalf("expected %q", wantErr)
	}
	if err != nil && err.Error() != wantErr {
		t.Fatalf("got %q, want %q", err, wantErr)
	}
	if err != nil {
		if _, ok := err.(*ValidationError); !ok {
			t.Fatalf("got error type %T, want *ValidationError", err)
		}
	}
}
