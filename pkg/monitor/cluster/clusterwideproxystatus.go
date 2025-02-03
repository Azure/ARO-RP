package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	cwp                  = "clusterWideProxy.status"
	cwpErrorMessage      = "NoProxy entries are incorrect"
	cluster              = "cluster"
	mandatory_no_proxies = "localhost,127.0.0.1,.svc,.cluster.local.,168.63.129.16"
	AzureDNS             = "169.254.169.254"
	//169.254.169.254 (the IMDS IP)
	//168.63.129.16 (Azure DNS, if no custom DNS exists)
	//localhost, 127.0.0.1, .svc, .cluster.local
)

// Helper function to emit and log the gauge status
func (mon *Monitor) emitAndLogCWPStatus(status bool, message string) {
	mon.emitGauge(cwp, 1, map[string]string{
		"status":  strconv.FormatBool(status),
		"Message": message,
	})

	if mon.hourlyRun {
		mon.log.WithFields(logrus.Fields{
			"metric":  cwp,
			"status":  strconv.FormatBool(status),
			"Message": message,
		}).Print()
	}
}

// Main function to emit CWP status
func (mon *Monitor) emitCWPStatus(ctx context.Context) error {
	mon.hourlyRun = true
	proxyConfig, err := mon.configcli.ConfigV1().Proxies().Get(ctx, cluster, metav1.GetOptions{})
	if err != nil {
		mon.log.Errorf("Error in getting the cluster wide proxy: %v", err)
		return err
	}
	if proxyConfig.Spec.HTTPProxy == "" && proxyConfig.Spec.HTTPSProxy == "" && proxyConfig.Spec.NoProxy == "" {
		mon.emitAndLogCWPStatus(false, "CWP not enabled")
	} else {
		// Create the noProxy map for efficient lookups
		no_proxy_list := strings.Split(proxyConfig.Spec.NoProxy, ",")
		noProxyMap := make(map[string]bool)
		var missing_no_proxy_list []string
		for _, proxy := range no_proxy_list {
			noProxyMap[proxy] = true
		}

		// Check mandatory no_proxy entries
		for _, mandatory_no_proxy := range strings.Split(mandatory_no_proxies, ",") {
			if !noProxyMap[mandatory_no_proxy] {
				missing_no_proxy_list = append(missing_no_proxy_list, mandatory_no_proxy)
			}
		}
		if !noProxyMap[AzureDNS] {
			dnsConfigcluster, err := mon.operatorcli.OperatorV1().DNSes().Get(ctx, "default", metav1.GetOptions{}) //mon.configcli.ConfigV1().DNSes().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				mon.log.Errorf("Error in getting DNS configuration: %v", err)
				return err
			}
			if len(dnsConfigcluster.Spec.Servers) == 0 {
				missing_no_proxy_list = append(missing_no_proxy_list, AzureDNS)
			}
		}

		mastersubnetID, err := azure.ParseResourceID(mon.oc.Properties.MasterProfile.SubnetID)
		if err != nil {
			mon.log.Errorf("failed to parse the mastersubnetID: %v", err)
			return err
		}
		token, err := mon.env.FPNewClientCertificateCredential(mon.tenantID, nil) //azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			mon.log.Errorf("failed to obtain FP Client Credentials: %v", err)
			return err
		}

		// Create client factory
		clientFactory, err := armnetwork.NewClientFactory(mastersubnetID.SubscriptionID, token, nil)
		if err != nil {
			mon.log.Errorf("failed to create client: %v", err)
			return err
		}

		// Check master subnet
		masterVnetID, _, err := apisubnet.Split(mon.oc.Properties.MasterProfile.SubnetID)
		if err != nil {
			mon.log.Errorf("failed to get the masterVnetID: %v", err)
			return err
		}
		mastervnetId, err := azure.ParseResourceID(masterVnetID)
		if err != nil {
			mon.log.Errorf("failed to parse the masterVnetID: %v", err)
			return err
		}
		res, err := clientFactory.NewSubnetsClient().Get(ctx, mastersubnetID.ResourceGroup, mastervnetId.ResourceName, mastersubnetID.ResourceName, &armnetwork.SubnetsClientGetOptions{Expand: nil})
		if err != nil {
			mon.log.Errorf("failed to finish the NewSubnetsClient request: %v", err)
			return err
		}

		if res.Properties.AddressPrefix != nil {
			if !noProxyMap[*res.Properties.AddressPrefix] {
				missing_no_proxy_list = append(missing_no_proxy_list, *res.Properties.AddressPrefix)
			}
		}

		// Check worker profiles
		for _, workerProfile := range mon.oc.Properties.WorkerProfiles {
			workersubnetID, err := azure.ParseResourceID(workerProfile.SubnetID)
			if err != nil {
				mon.log.Errorf("failed to parse the workersubnetID: %v", err)
				return err
			}
			workerVnetID, _, err := apisubnet.Split(workerProfile.SubnetID)
			if err != nil {
				mon.log.Errorf("failed to feth the workerVnetID: %v", err)
				return err
			}
			workervnetId, err := azure.ParseResourceID(workerVnetID)
			if err != nil {
				mon.log.Errorf("failed to parse the workerVnetID: %v", err)
				return err
			}
			workerres, err := clientFactory.NewSubnetsClient().Get(ctx, workersubnetID.ResourceGroup, workervnetId.ResourceName, workersubnetID.ResourceName, &armnetwork.SubnetsClientGetOptions{Expand: nil})
			if err != nil {
				mon.log.Errorf("failed to finish the request: %v", err)
			}
			if workerres.Properties.AddressPrefix != nil {
				workermachinesCIDR := *workerres.Properties.AddressPrefix
				if !noProxyMap[workermachinesCIDR] {
					missing_no_proxy_list = append(missing_no_proxy_list, workermachinesCIDR)
				}
			}
		}

		// Network Configuration Check
		networkConfig, err := mon.configcli.ConfigV1().Networks().Get(ctx, cluster, metav1.GetOptions{})
		if err != nil {
			mon.log.Errorf("Error in getting network info: %v", err)
			return err
		}
		for _, network := range networkConfig.Spec.ClusterNetwork {
			if !noProxyMap[network.CIDR] {
				missing_no_proxy_list = append(missing_no_proxy_list, network.CIDR)
			}
		}
		for _, network := range networkConfig.Spec.ServiceNetwork {
			if !noProxyMap[network] {
				missing_no_proxy_list = append(missing_no_proxy_list, network)
			}
		}

		// Gateway Domains Check
		clusterdetails, err := mon.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
		if err != nil {
			mon.log.Errorf("Error in getting cluster information: %v", err)
			return err
		}
		for _, gatewayDomain := range clusterdetails.Spec.GatewayDomains {
			if !noProxyMap[gatewayDomain] {
				missing_no_proxy_list = append(missing_no_proxy_list, gatewayDomain)
			}
		}

		// Infrastructure Configuration Check
		infraConfig, err := mon.configcli.ConfigV1().Infrastructures().Get(ctx, cluster, metav1.GetOptions{})
		if err != nil {
			mon.log.Errorf("Error in getting Infrasturcture info: %v", err)
			return err
		}

		// APIServerInternal URL Check
		apiServerIntURL, err := url.Parse(infraConfig.Status.APIServerInternalURL)
		if err != nil {
			mon.log.Errorf("Error in parsing APIServerProfile: %v", err)
			return err
		}
		apiServerIntdomain := strings.Split(apiServerIntURL.Host, ":")[0]
		if !noProxyMap[apiServerIntdomain] {
			missing_no_proxy_list = append(missing_no_proxy_list, apiServerIntdomain)
		}

		// APIServerProfile URL Check
		apiServerProfileURL, err := url.Parse(mon.oc.Properties.APIServerProfile.URL)
		if err != nil {
			mon.log.Errorf("Error in parsing APIServerProfile: %v", err)
			return err
		}
		apiServerProfiledomain := strings.Split(apiServerProfileURL.Host, ":")[0]
		if !noProxyMap[apiServerProfiledomain] {
			missing_no_proxy_list = append(missing_no_proxy_list, apiServerProfiledomain)
		}

		// ConsoleProfile URL Check
		consolProfileURL, err := url.Parse(mon.oc.Properties.ConsoleProfile.URL)
		if err != nil {
			mon.log.Errorf("Error in parsing ConsoleProfile: %v", err)
			return err
		}
		consoleProfiledomain := strings.Split(consolProfileURL.Host, ":")[0]
		if !noProxyMap[consolProfileURL.Host] {
			missing_no_proxy_list = append(missing_no_proxy_list, consoleProfiledomain)
		}
		if len(missing_no_proxy_list) > 0 {
			mon.emitAndLogCWPStatus(true, "CWP enabled but missing  "+strings.Join(missing_no_proxy_list, ",")+" in the no_proxy list")
		}
	}

	return nil
}
