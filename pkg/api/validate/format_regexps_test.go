package validate

import "testing"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestRxDomainName(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{
			value: "ok",
			want:  true,
		},
		{
			value: "8ad",
			want:  false,
		},
		{
			value: "ok.io",
			want:  true,
		},
		{
			value: "0k.io",
			want:  false,
		},
		{
			value: "lopadotemachoselachogaleokranioleipsanodrimhypotrimmatosilphioparaomelitokatakechymenokichlepikossyphophattoperisteralektryonoptekephalliokigklopeleiolagoiosiraiobaphetraganopterygon",
			want:  false,
		},
	} {
		t.Run(tt.value, func(t *testing.T) {
			if RxDomainName.MatchString(tt.value) != tt.want {
				t.Fatalf("%s didn't match %s", tt.value, RxDomainName)
			}
		})
	}
}
