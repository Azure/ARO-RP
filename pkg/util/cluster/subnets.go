package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"math/rand"
	"net"
)

// IPRangesOverlapCIDR reports whether any of ipRanges overlaps cidr.
func IPRangesOverlapCIDR(ipRanges []*net.IPNet, cidr string) (bool, error) {
	_, cidrNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	for _, snet := range ipRanges {
		if snet.Contains(cidrNet.IP) || cidrNet.Contains(snet.IP) {
			return true, nil
		}
	}
	return false, nil
}

// PickFreeDevVNetPrefix returns a random free /28 on the shared ARO
// dev-vnet, using the same address windows as generateSubnets:
//   - local (!isCI): 10.0.0.0/16
//   - CI (isCI):     10.3.0.0–10.127.x
func PickFreeDevVNetPrefix(used []*net.IPNet, prefixLen, maxTries int, isCI bool) (string, error) {
	if prefixLen != 28 {
		return "", fmt.Errorf("only /28 is supported, got /%d", prefixLen)
	}

	for i := 0; i < maxTries; i++ {
		// /28 network address must be a multiple of 16.
		host := rand.Intn(16) * 16

		var candidate string
		if !isCI {
			candidate = fmt.Sprintf("10.0.%d.%d/28", rand.Intn(256), host)
		} else {
			candidate = fmt.Sprintf("10.%d.%d.%d/28", 3+rand.Intn(125), rand.Intn(256), host)
		}

		overlap, err := IPRangesOverlapCIDR(used, candidate)
		if err != nil || overlap {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("could not find free /28 after %d tries", maxTries)
}
