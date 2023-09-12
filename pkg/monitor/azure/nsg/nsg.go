package nsg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
)

const (
	MetricInvalidDenyRule          = "monitor.preconfigurednsg.invaliddenyrule"
	MetricSubnetAccessResponseCode = "monitor.preconfigurednsg.subnetaccessresponsecode"
	MetricSubnetAccessForbidden    = "monitor.preconfigurednsg.subnetaccessforbidden"
	MetricUnsuccessfulFPCreation   = "monitor.preconfigurednsg.fpcreationunsuccessful"
	MetricNSGMonitoringTimedOut    = "monitor.preconfigurednsg.monitoringtimedout"
)

var expandNSG = "NetworkSecurityGroup"

// NSGMonitor is responsible for performing NSG rule validations when preconfiguredNSG is enabled
type NSGMonitor struct {
	log     *logrus.Entry
	emitter metrics.Emitter
	oc      *api.OpenShiftCluster

	subnetClient *armnetwork.SubnetsClient
	done         chan error
	dims         map[string]string
}

func (n *NSGMonitor) Done() <-chan error {
	return n.done
}

func NewNSGMonitor(log *logrus.Entry, oc *api.OpenShiftCluster, subscriptionID string, subnetClient *armnetwork.SubnetsClient, emitter metrics.Emitter) *NSGMonitor {
	return &NSGMonitor{
		log:     log,
		emitter: emitter,
		oc:      oc,

		subnetClient: subnetClient,
		done:         make(chan error),

		dims: map[string]string{
			dimension.ResourceID:     oc.ID,
			dimension.SubscriptionID: subscriptionID,
		},
	}
}

type subnetNSGConfig struct {
	// subnet CIDR range
	prefix []netip.Prefix
	// The rules from the subnet NSG
	nsg *armnetwork.SecurityGroup
}

func (n *NSGMonitor) toSubnetConfig(ctx context.Context, subnetID string) (subnetNSGConfig, error) {
	r, err := arm.ParseResourceID(subnetID)
	if err != nil {
		return subnetNSGConfig{}, err
	}

	dims := map[string]string{
		dimension.ResourceID:     n.oc.ID,
		dimension.Location:       n.oc.Location,
		dimension.Subnet:         r.Name,
		dimension.Vnet:           r.Parent.Name,
		dimension.ResourceGroup:  r.ResourceGroupName,
		dimension.SubscriptionID: r.SubscriptionID,
	}

	subnet, err := n.subnetClient.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, &armnetwork.SubnetsClientGetOptions{Expand: &expandNSG})
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr); respErr.StatusCode == http.StatusForbidden {
			n.emitter.EmitGauge(MetricSubnetAccessForbidden, int64(1), dims)
		}
		n.log.Errorf("error while getting subnet %s. %s", subnetID, err)
		return subnetNSGConfig{}, err
	}

	var cidrs []string
	if subnet.Properties.AddressPrefix != nil {
		cidrs = append(cidrs, *subnet.Properties.AddressPrefix)
	}
	for _, sn := range subnet.Properties.AddressPrefixes {
		cidrs = append(cidrs, *sn)
	}
	prefixes := toPrefixes(n.log, cidrs)
	if len(prefixes) == 0 {
		return subnetNSGConfig{}, errors.New("no valid subnet ranges found")
	}
	return subnetNSGConfig{prefixes, subnet.Properties.NetworkSecurityGroup}, nil
}

func (n *NSGMonitor) Monitor(ctx context.Context) {
	masterSubnet, err := n.toSubnetConfig(ctx, n.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		// FP has no access to the subnet
		n.done <- err
		return
	}

	// need this to get the right workerProfiles
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(n.oc.Properties)
	workerSubnets := make([]subnetNSGConfig, 0, len(workerProfiles))
	workerPrefixes := make([]netip.Prefix, 0, len(workerProfiles))
	for _, wp := range workerProfiles {
		// Customer can configure a machineset with an invalid subnet.
		// In such case, the subnetID will be empty.
		if len(wp.SubnetID) == 0 {
			continue
		}

		s, err := n.toSubnetConfig(ctx, wp.SubnetID)
		if err != nil {
			// FP has no access to the subnet
			n.done <- err
			return
		}
		workerSubnets = append(workerSubnets, s)
		workerPrefixes = append(workerPrefixes, s.prefix...)
	}

	// to make sure each NSG is processed only once
	nsgSet := map[string]*armnetwork.SecurityGroup{}
	if masterSubnet.nsg != nil && masterSubnet.nsg.ID != nil {
		nsgSet[*masterSubnet.nsg.ID] = masterSubnet.nsg

	}
	for _, w := range workerSubnets {
		if w.nsg != nil && w.nsg.ID != nil {
			nsgSet[*w.nsg.ID] = w.nsg
		}
	}

	for nsgID, nsg := range nsgSet {
		for _, rule := range nsg.Properties.SecurityRules {
			if rule.Properties.Access != nil && *rule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
				// Allow rule - skip.
				continue
			}
			// Deny rule
			nsgResource, err := arm.ParseResourceID(nsgID)
			if err != nil {
				n.log.Errorf("Unable to parse NSG resource ID: %s. %s", nsgID, err)
				continue
			}

			r := newRuleChecker(n.log, masterSubnet.prefix, workerPrefixes, rule)

			if r.isInvalidDenyRule() {
				dims := map[string]string{
					dimension.ResourceID:          n.oc.ID,
					dimension.Location:            n.oc.Location,
					dimension.SubscriptionID:      nsgResource.SubscriptionID,
					dimension.ResourceGroup:       nsgResource.ResourceGroupName,
					dimension.ResourceName:        nsgResource.Name,
					dimension.NSGRuleName:         *rule.Name,
					dimension.NSGRuleSources:      strings.Join(r.sourceStrings, ","),
					dimension.NSGRuleDestinations: strings.Join(r.destinationStrings, ","),
					dimension.NSGRuleDirection:    string(*rule.Properties.Direction),
					dimension.NSGRulePriority:     string(*rule.Properties.Priority),
				}
				n.emitter.EmitGauge(MetricInvalidDenyRule, int64(1), dims)
			}
		}
	}
	n.done <- nil
}
