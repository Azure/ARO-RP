package nsg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/netip"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

const (
	virtualNetwork = "VirtualNetwork"
	wildcard       = "*"
)

type ruleChecker struct {
	log                               *logrus.Entry
	master                            []netip.Prefix
	workers                           []netip.Prefix
	rule                              *armnetwork.SecurityRule
	sourceStrings, destinationStrings []string
}

func newRuleChecker(log *logrus.Entry, master []netip.Prefix, workers []netip.Prefix, rule *armnetwork.SecurityRule) ruleChecker {
	var sourceStrings, destinationStrings []string
	if rule.Properties.SourceAddressPrefix != nil {
		sourceStrings = append(sourceStrings, *rule.Properties.SourceAddressPrefix)
	}
	srcPrefixes := rule.Properties.SourceAddressPrefixes
	for i := range srcPrefixes {
		sourceStrings = append(sourceStrings, *srcPrefixes[i])
	}
	if rule.Properties.DestinationAddressPrefix != nil {
		destinationStrings = append(destinationStrings, *rule.Properties.DestinationAddressPrefix)
	}
	destPrefixes := rule.Properties.DestinationAddressPrefixes
	for i := range destPrefixes {
		destinationStrings = append(destinationStrings, *destPrefixes[i])
	}
	return ruleChecker{
		log:                log,
		master:             master,
		workers:            workers,
		rule:               rule,
		sourceStrings:      sourceStrings,
		destinationStrings: destinationStrings,
	}
}

// For storing evaluation results
type properties struct {
	isMaster         bool
	isWorker         bool
	isAny            bool
	isVirtualNetwork bool
}

func (p properties) isNothing() bool {
	return !p.isMaster && !p.isWorker && !p.isAny && !p.isVirtualNetwork
}

func (c ruleChecker) newProperties(addresses []string) properties {
	return properties{
		isMaster:         c.isMaster(addresses),
		isWorker:         c.isWorker(addresses),
		isAny:            isAny(addresses),
		isVirtualNetwork: isVirtualNetwork(addresses),
	}
}

func (c ruleChecker) isInvalidDenyRule() bool {
	source := c.newProperties(c.sourceStrings)
	destination := c.newProperties(c.destinationStrings)

	switch {
	case source.isNothing() && destination.isNothing():
		return false
	case source.isMaster && destination.isMaster:
		return true
	case source.isMaster && destination.isWorker:
		return true
	case source.isWorker && destination.isMaster:
		return true
	case source.isMaster && destination.isAny:
		return true
	case source.isAny && destination.isMaster:
		return true
	case source.isWorker && destination.isAny:
		return true
	case source.isAny && destination.isWorker:
		return true
	case source.isAny && destination.isAny:
		return true
	case source.isMaster && destination.isVirtualNetwork:
		return true
	case source.isVirtualNetwork && destination.isMaster:
		return true
	case source.isWorker && destination.isVirtualNetwork:
		return true
	case source.isVirtualNetwork && destination.isWorker:
		return true
	case source.isVirtualNetwork && destination.isAny:
		return true
	case source.isAny && destination.isVirtualNetwork:
		return true
	case source.isVirtualNetwork && destination.isVirtualNetwork:
		return true
	}
	return false
}

func (c ruleChecker) isMaster(addresses []string) bool {
	prefixes := toPrefixes(c.log, addresses)
	return overlaps(prefixes, c.master)
}

func (c ruleChecker) isWorker(addresses []string) bool {
	prefixes := toPrefixes(c.log, addresses)
	return overlaps(prefixes, c.workers)
}

func overlaps(prefixes []netip.Prefix, subnet []netip.Prefix) bool {
	for _, subnetcidr := range subnet {
		for _, p := range prefixes {
			if subnetcidr.Overlaps(p) {
				return true
			}
		}
	}
	return false
}

func isAny(addresses []string) bool {
	for _, address := range addresses {
		if address == wildcard {
			return true
		}
	}
	return false
}

func isVirtualNetwork(addresses []string) bool {
	for _, address := range addresses {
		if address == virtualNetwork {
			return true
		}
	}
	return false
}

// toPrefix converts a string value of an IP address or a CIDR range into a prefix
func toPrefix(value string) (netip.Prefix, error) {
	if addr, err := netip.ParseAddr(value); err == nil {
		return netip.PrefixFrom(addr, 32), nil
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix, nil
	}
	return netip.Prefix{}, fmt.Errorf("invalid IP address or CIDR range: %s", value)
}

// toPrefixes converts a slice of addresses into a slice of Prefixes
func toPrefixes(log *logrus.Entry, addresses []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(addresses))
	for _, address := range addresses {
		prefix, err := toPrefix(address)
		if err != nil {
			// can be continued safely to the next one because:
			//  1. The strings always come directly from Azure, which has been validated.
			//  2. Even if the value is wrong, it won't be neither master or worker.
			//  3. We should also skip other service tags (VirtualNetwork, Internet, Any etc)
			log.Debugf("Error while parsing %s. Full error %s.", address, err)
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}
