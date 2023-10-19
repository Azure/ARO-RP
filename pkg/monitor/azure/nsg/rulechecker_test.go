package nsg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/netip"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestToPrefix(t *testing.T) {
	for _, tt := range []struct {
		name    string
		in      string
		want    netip.Prefix
		wantErr string
	}{
		{
			name: "pass - valid CIDR range",
			in:   "10.0.0.0/24",
			want: netip.MustParsePrefix("10.0.0.0/24"),
		},
		{
			name:    "failed - invalid CIDR range",
			in:      "10.0.0.0/33",
			want:    netip.Prefix{},
			wantErr: `invalid IP address or CIDR range: 10.0.0.0/33`,
		},
		{
			name: "pass - valid IP address",
			in:   "10.0.0.1",
			want: netip.PrefixFrom(netip.MustParseAddr("10.0.0.1"), 32),
		},
		{
			name:    "failed - invalid IP address",
			in:      "300.0.0.1",
			want:    netip.Prefix{},
			wantErr: `invalid IP address or CIDR range: 300.0.0.1`,
		},
		{
			name:    "failed - blank",
			in:      "",
			want:    netip.Prefix{},
			wantErr: `invalid IP address or CIDR range: `,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toPrefix(tt.in)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if got != tt.want {
				t.Errorf("getPrefix got %s want %s", got, tt.want)
			}
		})
	}
}

func TestToPrefixes(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	for _, tt := range []struct {
		name string
		in   []string
		want []netip.Prefix
	}{
		{
			name: "pass - valid IPs only",
			in:   []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			want: []netip.Prefix{
				netip.MustParsePrefix("10.0.0.1/32"),
				netip.MustParsePrefix("10.0.0.2/32"),
				netip.MustParsePrefix("10.0.0.3/32"),
			},
		},
		{
			name: "pass - valid CIDRs only",
			in:   []string{"10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/24"},
			want: []netip.Prefix{
				netip.MustParsePrefix("10.0.0.0/24"),
				netip.MustParsePrefix("10.0.1.0/24"),
				netip.MustParsePrefix("10.0.2.0/24"),
			},
		},
		{
			name: "pass - combination of IPs and CIDRs",
			in:   []string{"10.0.0.0/24", "10.0.1.0", "10.0.2.0/24", "10.0.3.0"},
			want: []netip.Prefix{
				netip.MustParsePrefix("10.0.0.0/24"),
				netip.MustParsePrefix("10.0.1.0/32"),
				netip.MustParsePrefix("10.0.2.0/24"),
				netip.MustParsePrefix("10.0.3.0/32"),
			},
		},
		{
			name: "pass - skip invalid IPs and CIDRs",
			in:   []string{"10.0.0.0/24", "1000.0.1.0", "1000.0.2.0/24", "10.0.3.0"},
			want: []netip.Prefix{
				netip.MustParsePrefix("10.0.0.0/24"),
				netip.MustParsePrefix("10.0.3.0/32"),
			},
		},
		{
			name: "pass - blank",
			in:   []string{},
			want: []netip.Prefix{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := toPrefixes(log, tt.in)
			if len(got) != len(tt.want) {
				t.Errorf("want result length %d, got %d", len(tt.want), len(got))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("want %s got %s on index %d", tt.want[i], got[i], i)
				}
			}
		})
	}
}

func TestIsVirtualNetwork(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   []string
		want bool
	}{
		{
			name: "true",
			in:   []string{"this is not", "VirtualNetwork"},
			want: true,
		},
		{
			name: "false - no VirtualNetwork",
			in:   []string{"this is not", "vNet"},
			want: false,
		},
		{
			name: "false - blank",
			in:   []string{},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := isVirtualNetwork(tt.in)
			if got != tt.want {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}

func TestIsAny(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   []string
		want bool
	}{
		{
			name: "true",
			in:   []string{"this is not", "*"},
			want: true,
		},
		{
			name: "false - no VirtualNetwork",
			in:   []string{"this is not", "vNet"},
			want: false,
		},
		{
			name: "false - blank",
			in:   []string{},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := isAny(tt.in)
			if got != tt.want {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}

func TestRuleCheckerIsMaster(t *testing.T) {
	generator := func(c *ruleChecker) {
		c.master = []netip.Prefix{
			netip.MustParsePrefix("10.0.0.0/24"),
		}
	}
	for _, tt := range []struct {
		name string
		mod  func(*ruleChecker)
		in   []string
		want bool
	}{
		{
			name: "true - overlapping",
			mod:  generator,
			in: []string{
				"10.0.0.1/32",
				"192.168.0.0/24",
			},
			want: true,
		},
		{
			name: "false - non-overlapping",
			mod:  generator,
			in: []string{
				"172.28.0.1/32",
				"192.168.0.0/24",
			},
			want: false,
		},
		{
			name: "false - not overlapping with nothing",
			mod:  generator,
			in:   []string{},
			want: false,
		},
		{
			name: "false - uninitialized checker",
			mod: func(c *ruleChecker) {
				// set nothing
			},
			in:   []string{},
			want: false,
		},
		{
			name: "true - overlapping one of multiple prefixes",
			mod: func(c *ruleChecker) {
				generator(c)
				c.master = append(c.master, netip.MustParsePrefix("10.0.1.0/24"))
			},
			in: []string{
				"10.0.0.1/32",
				"192.168.0.0/24",
			},
			want: true,
		},
		{
			name: "true - overlapping all of the prefixes",
			mod: func(c *ruleChecker) {
				generator(c)
				c.master = append(c.master, netip.MustParsePrefix("10.0.1.0/24"))
			},
			in: []string{
				"10.0.0.1/32",
				"10.0.1.1/32",
			},
			want: true,
		},
		{
			name: "false - not overlapping with any of the multiple prefixes",
			mod: func(c *ruleChecker) {
				generator(c)
				c.master = append(c.master, netip.MustParsePrefix("10.0.1.0/24"))
			},
			in: []string{
				"10.1.0.1/32",
				"10.1.1.1/32",
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := ruleChecker{}
			tt.mod(&c)
			got := c.isMaster(tt.in)
			if got != tt.want {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}

func TestRuleCheckerIsWorker(t *testing.T) {
	generator := func(c *ruleChecker) {
		c.workers = []netip.Prefix{
			netip.MustParsePrefix("10.0.0.0/24"),
			netip.MustParsePrefix("10.0.1.0/24"),
		}
	}
	for _, tt := range []struct {
		name string
		mod  func(*ruleChecker)
		in   []string
		want bool
	}{
		{
			name: "true - overlapping",
			mod:  generator,
			in: []string{
				"10.0.0.1/32",
				"192.168.0.0/24",
			},
			want: true,
		},
		{
			name: "false - non-overlapping",
			mod:  generator,
			in: []string{
				"172.28.0.1/32",
				"192.168.0.0/24",
			},
			want: false,
		},
		{
			name: "false - not overlapping with nothing",
			mod:  generator,
			in:   []string{},
			want: false,
		},
		{
			name: "false - uninitialized checker",
			mod: func(c *ruleChecker) {
				// set nothing
			},
			in:   []string{},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := ruleChecker{}
			tt.mod(&c)
			got := c.isWorker(tt.in)
			if got != tt.want {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}

func TestPropertiesIsNothing(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   properties
		want bool
	}{
		{
			name: "nothing",
			in:   properties{false, false, false, false},
			want: true,
		},
		{
			name: "something - isMaster",
			in:   properties{true, false, false, false},
			want: false,
		},
		{
			name: "all",
			in:   properties{true, true, true, true},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.isNothing()
			if got != tt.want {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}

func TestRuleCheckerIsInvalidDenyRule(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	masterPrefixes := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/24"),
		netip.MustParsePrefix("11.0.0.0/24"),
	}
	workerPrefixes := []netip.Prefix{
		netip.MustParsePrefix("10.0.1.0/24"),
		netip.MustParsePrefix("10.0.2.0/24"),
	}
	var (
		nothing = "172.28.0.0/24"
		master  = "10.0.0.1/32"

		master1 = "10.0.0.1/32"
		master2 = "10.0.0.2/32"
		masters = []*string{&master1, &master2}

		worker = "10.0.1.1/32"

		worker1  = "10.0.1.1/32"
		worker2  = "10.0.1.2/32"
		workers  = []*string{&worker1, &worker2}
		wildcard = "*"
		vnet     = "VirtualNetwork"
	)
	for _, tt := range []struct {
		name string
		rule armnetwork.SecurityRule
		want bool
	}{
		{
			name: "valid - both nothing",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &nothing,
					DestinationAddressPrefix: &nothing,
				},
			},
			want: false,
		},
		{
			name: "valid - source master, destination nothing",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &master,
					DestinationAddressPrefix: &nothing,
				},
			},
			want: false,
		},
		{
			name: "valid - source master (prefixes), destination nothing",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefixes:    masters,
					DestinationAddressPrefix: &nothing,
				},
			},
			want: false,
		},
		{
			name: "valid - source nothing, destination master",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &nothing,
					DestinationAddressPrefix: &master,
				},
			},
			want: false,
		},
		{
			name: "invalid - source master, destination master",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &master,
					DestinationAddressPrefix: &master,
				},
			},
			want: true,
		},
		{
			name: "invalid - source master, destination worker",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &master,
					DestinationAddressPrefix: &worker,
				},
			},
			want: true,
		},
		{
			name: "invalid - source worker (prefixes), destination master",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefixes:    workers,
					DestinationAddressPrefix: &master,
				},
			},
			want: true,
		},
		{
			name: "valid - source worker (prefixes), destination worker",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefixes:    workers,
					DestinationAddressPrefix: &worker,
				},
			},
			want: false,
		},
		{
			name: "invalid - source master (prefixes), destination any",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefixes:    masters,
					DestinationAddressPrefix: &wildcard,
				},
			},
			want: true,
		},
		{
			name: "invalid - source workers (prefixes), destination any",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefixes:    workers,
					DestinationAddressPrefix: &wildcard,
				},
			},
			want: true,
		},
		{
			name: "invalid - source any, destination master",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &wildcard,
					DestinationAddressPrefix: &master,
				},
			},
			want: true,
		},
		{
			name: "invalid - source any, destination worker (prefixes)",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:        &wildcard,
					DestinationAddressPrefixes: workers,
				},
			},
			want: true,
		},
		{
			name: "invalid - source any, destination any",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &wildcard,
					DestinationAddressPrefix: &wildcard,
				},
			},
			want: true,
		},
		{
			name: "invalid - source master, destination vnet",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &master,
					DestinationAddressPrefix: &vnet,
				},
			},
			want: true,
		},
		{
			name: "invalid - source vnet, destination master (prefixes)",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:        &vnet,
					DestinationAddressPrefixes: masters,
				},
			},
			want: true,
		},
		{
			name: "invalid - source worker, destination vnet",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &worker,
					DestinationAddressPrefix: &vnet,
				},
			},
			want: true,
		},
		{
			name: "invalid - source vnet, destination worker",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &vnet,
					DestinationAddressPrefix: &worker,
				},
			},
			want: true,
		},
		{
			name: "invalid - source vnet, destination any",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &vnet,
					DestinationAddressPrefix: &wildcard,
				},
			},
			want: true,
		},
		{
			name: "invalid - source any, destination vnet",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &wildcard,
					DestinationAddressPrefix: &vnet,
				},
			},
			want: true,
		},
		{
			name: "invalid - source vnet, destination vnet",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &vnet,
					DestinationAddressPrefix: &vnet,
				},
			},
			want: true,
		},
		{
			name: "valid - source nothing, destination vnet",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &nothing,
					DestinationAddressPrefix: &vnet,
				},
			},
			want: false,
		},
		{
			name: "valid - source vnet, destination nothing",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &vnet,
					DestinationAddressPrefix: &nothing,
				},
			},
			want: false,
		},
		{
			name: "valid - source any, destination nothing",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &wildcard,
					DestinationAddressPrefix: &nothing,
				},
			},
			want: false,
		},
		{
			name: "valid - source nothing, destination any",
			rule: armnetwork.SecurityRule{
				Properties: &armnetwork.SecurityRulePropertiesFormat{
					SourceAddressPrefix:      &nothing,
					DestinationAddressPrefix: &wildcard,
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newRuleChecker(log, masterPrefixes, workerPrefixes, &tt.rule)
			got := c.isInvalidDenyRule()
			if got != tt.want {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}
