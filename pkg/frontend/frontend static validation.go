package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"net"
)

// SubnetCIDR checks if the given IP net is a valid CIDR.
func SubnetCIDR(cidr *net.IPNet) error {
	if cidr.IP.IsUnspecified() {
		return errors.New("address must be specified")
	}
	nip := cidr.IP.Mask(cidr.Mask)
	if nip.String() != cidr.IP.String() {
		return fmt.Errorf("invalid network address. got %s, expecting %s", cidr.String(), (&net.IPNet{IP: nip, Mask: cidr.Mask}).String())
	}
	return nil
}
