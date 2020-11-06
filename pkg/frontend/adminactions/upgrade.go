package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/status"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (a *adminactions) Upgrade(ctx context.Context, upgradeY bool) error {
	err := preUpgradeChecks(ctx, a.oc, a.vNetClient)
	if err != nil {
		return err
	}

	return upgrade(ctx, a.log, a.configClient, version.Streams, upgradeY)
}

func upgrade(ctx context.Context, log *logrus.Entry, configClient configclient.Interface, streams []*version.Stream, upgradeY bool) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := configClient.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// We don't upgrade if cluster upgrade is not finished
		if !status.ClusterVersionOperatorIsHealthy(cv.Status) {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Not upgrading: cvo is unhealthy.")
		}

		desired, err := version.ParseVersion(cv.Status.Desired.Version)
		if err != nil {
			return err
		}

		// Get Cluster upgrade version based on desired version
		stream := version.GetUpgradeStream(streams, desired, upgradeY)
		if stream == nil {
			log.Info("not upgrading: stream not found")
			return nil
		}

		log.Printf("initiating cluster upgrade, target version %s", stream.Version.String())

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: stream.Version.String(),
			Image:   stream.PullSpec,
		}

		_, err = configClient.ConfigV1().ClusterVersions().Update(ctx, cv, metav1.UpdateOptions{})
		return err
	})
}

func preUpgradeChecks(ctx context.Context, oc *api.OpenShiftCluster, vnet network.VirtualNetworksClient) error {
	return checkCustomDNS(ctx, oc, vnet)
}

// checkCustomDNS checks if customer has custom DNS configured on VNET.
// This would cause nodes to rotate and render cluster inoperable
func checkCustomDNS(ctx context.Context, oc *api.OpenShiftCluster, vnet network.VirtualNetworksClient) error {
	infraID := oc.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	vnetID, _, err := subnet.Split(oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	v, err := vnet.Get(ctx, r.ResourceGroup, r.ResourceName, "")
	if err != nil {
		return err
	}

	if v.VirtualNetworkPropertiesFormat.DhcpOptions != nil &&
		v.VirtualNetworkPropertiesFormat.DhcpOptions.DNSServers != nil &&
		len(*v.VirtualNetworkPropertiesFormat.DhcpOptions.DNSServers) > 0 {
		return fmt.Errorf("not upgrading: custom DNS is set")
	}

	return nil
}
