package cmp

import (
	"crypto/x509"
	"testing"
)

func TestX509CertComparer(t *testing.T) {
	tests := []struct {
		name   string
		x, y   *x509.Certificate
		expect bool
	}{
		{
			name:   "both nil",
			x:      nil,
			y:      nil,
			expect: true,
		},
		{
			name:   "one nil: x",
			x:      nil,
			y:      &x509.Certificate{},
			expect: false,
		},
		{
			name:   "one nil: y",
			x:      &x509.Certificate{},
			y:      nil,
			expect: false,
		},
		{
			name:   "all non-nil and equal",
			x:      &x509.Certificate{Raw: []byte{1}},
			y:      &x509.Certificate{Raw: []byte{1}},
			expect: true,
		},
		{
			name:   "all non-nil and not equal",
			x:      &x509.Certificate{Raw: []byte{1}},
			y:      &x509.Certificate{Raw: []byte{2}},
			expect: false,
		},
	}

	for _, test := range tests {
		got := x509CertComparer(test.x, test.y)
		if got != test.expect {
			t.Errorf("%s: expected %#v got %#v", test.name, test.expect, got)
		}
	}
}
