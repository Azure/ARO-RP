package nsg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/monitor/emitter"
	"github.com/Azure/ARO-RP/pkg/monitor/monitoring"
	sdknetwork "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
)

const (
	MetricPreconfiguredNSGEnabled  = "monitor.preconfigurednsg.enabled"
	MetricFailedNSGMonitorCreation = "monitor.preconfigurednsg.failedmonitorcreation"
	MetricInvalidDenyRule          = "monitor.preconfigurednsg.invaliddenyrule"
	MetricSubnetAccessForbidden    = "monitor.preconfigurednsg.subnetaccessforbidden"
	MetricSubnetAccessResponseCode = "monitor.preconfigurednsg.subnetaccessresponsecode"
)

var expandNSG = "NetworkSecurityGroup"

var _ monitoring.Monitor = (*NSGMonitor)(nil)

// NSGMonitor is responsible for performing NSG rule validations when preconfiguredNSG is enabled
type NSGMonitor struct {
	log     *logrus.Entry
	emitter metrics.Emitter
	oc      *api.OpenShiftCluster

	wg *sync.WaitGroup

	subnetClient sdknetwork.SubnetsClient
	dims         map[string]string
}

func NewMonitor(log *logrus.Entry, oc *api.OpenShiftCluster, e env.Interface, subscriptionID string, tenantID string, emitter metrics.Emitter, dims map[string]string, wg *sync.WaitGroup, trigger <-chan time.Time) monitoring.Monitor {
	if oc == nil {
		return &monitoring.NoOpMonitor{Wg: wg}
	}

	if oc.Properties.NetworkProfile.PreconfiguredNSG != api.PreconfiguredNSGEnabled {
		return &monitoring.NoOpMonitor{Wg: wg}
	}

	emitter.EmitGauge(MetricPreconfiguredNSGEnabled, int64(1), dims)

	select {
	case <-trigger:
	default:
		return &monitoring.NoOpMonitor{Wg: wg}
	}

	token, err := e.FPNewClientCertificateCredential(tenantID)
	if err != nil {
		log.Error("Unable to create FP Authorizer for NSG monitoring.", err)
		emitter.EmitGauge(MetricFailedNSGMonitorCreation, int64(1), dims)
		return &monitoring.NoOpMonitor{Wg: wg}
	}

	clientOptions := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: e.Environment().Cloud,
		},
	}

	client, err := sdknetwork.NewSubnetsClient(subscriptionID, token, &clientOptions)
	if err != nil {
		log.Error("Unable to create the subnet client for NSG monitoring", err)
		emitter.EmitGauge(MetricFailedNSGMonitorCreation, int64(1), dims)
		return &monitoring.NoOpMonitor{Wg: wg}
	}

	return &NSGMonitor{
		log:     log,
		emitter: emitter,
		oc:      oc,

		subnetClient: client,
		wg:           wg,

		dims: dims,
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
		dimension.Subnet:           r.Name,
		dimension.Vnet:             r.Parent.Name,
		dimension.NSGResourceGroup: r.ResourceGroupName,
	}

	subnet, err := n.subnetClient.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, &armnetwork.SubnetsClientGetOptions{Expand: &expandNSG})
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr); respErr.StatusCode == http.StatusForbidden {
			emitter.EmitGauge(n.emitter, MetricSubnetAccessForbidden, int64(1), n.dims, dims)
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

// Monitor checks the custom NSGs customers attach to their ARO subnets
func (n *NSGMonitor) Monitor(ctx context.Context) []error {
	defer n.wg.Done()

	errors := []error{}

	// to make sure each NSG is processed only once
	nsgSet := map[string]*armnetwork.SecurityGroup{}

	masterSubnet, err := n.toSubnetConfig(ctx, n.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		// FP has no access to the subnet
		errors = append(errors, err)
	} else {
		if masterSubnet.nsg != nil && masterSubnet.nsg.ID != nil {
			nsgSet[*masterSubnet.nsg.ID] = masterSubnet.nsg
		}
	}

	// need this to get the right workerProfiles
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(n.oc.Properties)
	workerPrefixes := make([]netip.Prefix, 0, len(workerProfiles))
	// To minimize the possibility of NRP throttling, we only retrieve a subnet's info only once.
	subnetsToMonitor := map[string]struct{}{}

	for _, wp := range workerProfiles {
		// Customer can configure a machineset with an invalid subnet.
		// In such case, the subnetID will be empty.
		// Also 0 machine means no worker nodes are there, so no point to monitor the profile.
		if len(wp.SubnetID) != 0 && wp.Count != 0 {
			subnetsToMonitor[wp.SubnetID] = struct{}{}
		}
	}

	for subnetID := range subnetsToMonitor {
		s, err := n.toSubnetConfig(ctx, subnetID)
		if err != nil {
			// FP has no access to the subnet
			errors = append(errors, err)
		} else {
			workerPrefixes = append(workerPrefixes, s.prefix...)
			if s.nsg != nil && s.nsg.ID != nil {
				nsgSet[*s.nsg.ID] = s.nsg
			}
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
				errors = append(errors, err)
				continue
			}

			r := newRuleChecker(n.log, masterSubnet.prefix, workerPrefixes, rule)

			if r.isInvalidDenyRule() {
				dims := map[string]string{
					dimension.NSGResourceGroup:    nsgResource.ResourceGroupName,
					dimension.NSG:                 nsgResource.Name,
					dimension.NSGRuleName:         *rule.Name,
					dimension.NSGRuleSources:      strings.Join(r.sourceStrings, ","),
					dimension.NSGRuleDestinations: strings.Join(r.destinationStrings, ","),
					dimension.NSGRuleDirection:    string(*rule.Properties.Direction),
					dimension.NSGRulePriority:     fmt.Sprint(*rule.Properties.Priority),
				}
				emitter.EmitGauge(n.emitter, MetricInvalidDenyRule, int64(1), n.dims, dims)
			}
		}
	}
	return errors
}
