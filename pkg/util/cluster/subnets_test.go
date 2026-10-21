package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net"
	"strings"
	"testing"
)

func TestIPRangesOverlapCIDR(t *testing.T) {
	_, used, err := net.ParseCIDR("10.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}

	overlap, err := IPRangesOverlapCIDR([]*net.IPNet{used}, "10.0.0.128/28")
	if err != nil {
		t.Fatal(err)
	}
	if !overlap {
		t.Fatal("expected overlap with containing /24")
	}

	overlap, err = IPRangesOverlapCIDR([]*net.IPNet{used}, "10.0.1.0/28")
	if err != nil {
		t.Fatal(err)
	}
	if overlap {
		t.Fatal("expected no overlap with adjacent /28")
	}
}

func TestPickFreeDevVNetPrefix(t *testing.T) {
	_, used, err := net.ParseCIDR("10.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}

	got, err := PickFreeDevVNetPrefix([]*net.IPNet{used}, 28, GenerateSubnetMaxTries, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "10.0.") || !strings.HasSuffix(got, "/28") {
		t.Fatalf("local pick = %q, want 10.0.x.x/28", got)
	}
	overlap, err := IPRangesOverlapCIDR([]*net.IPNet{used}, got)
	if err != nil {
		t.Fatal(err)
	}
	if overlap {
		t.Fatalf("picked overlapping prefix %q", got)
	}

	got, err = PickFreeDevVNetPrefix(nil, 28, GenerateSubnetMaxTries, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "/28") {
		t.Fatalf("CI pick = %q, want /28", got)
	}
	_, cidr, err := net.ParseCIDR(got)
	if err != nil {
		t.Fatal(err)
	}
	second := int(cidr.IP.To4()[1])
	if second < 3 || second > 127 {
		t.Fatalf("CI pick second octet = %d, want [3,127]", second)
	}
}
