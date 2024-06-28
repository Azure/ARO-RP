package net

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest/azure"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	v1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Listen returns a listener with its send and receive buffer sizes set, such
// that sockets which are *returned* by the listener when Accept() is called
// also have those buffer sizes.
func Listen(network, address string, sz int) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	sc, ok := l.(syscall.Conn)
	if !ok {
		return nil, errors.New("listener does not implement Syscall.Conn")
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return nil, err
	}

	err = setBuffers(rc, sz)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// Dial returns a dialled connection with its send and receive buffer sizes set.
// If sz <= 0, we leave the default size.
func Dial(network, address string, sz int) (net.Conn, error) {
	return (&net.Dialer{
		Control: func(network, address string, rc syscall.RawConn) error {
			if sz <= 0 {
				return nil
			}

			return setBuffers(rc, sz)
		},
	}).Dial(network, address)
}

// read socket(7)
func setBuffers(rc syscall.RawConn, sz int) error {
	var err2 error
	err := rc.Control(func(fd uintptr) {
		err2 = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, sz)
	})
	if err2 != nil {
		return err2
	}
	if err != nil {
		return err
	}

	err = rc.Control(func(fd uintptr) {
		err2 = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, sz)
	})
	if err2 != nil {
		return err2
	}

	return err
}

type PrivateZoneRemovalConfig struct {
	Log                *logrus.Entry
	PrivateZonesClient privatedns.PrivateZonesClient
	Configcli          configclient.Interface
	Mcocli             mcoclient.Interface
	Kubernetescli      kubernetes.Interface
	VNetLinksClient    privatedns.VirtualNetworkLinksClient
	ResourceGroupID    string
}

func RemovePrivateDNSZone(ctx context.Context, config PrivateZoneRemovalConfig) error {
	if config.PrivateZonesClient == nil {
		return errors.New("privateZonesClient is nil")
	}

	resourceGroup := stringutils.LastTokenByte(config.ResourceGroupID, '/')
	privateZones, err := config.PrivateZonesClient.ListByResourceGroup(ctx, resourceGroup, nil)
	if err != nil {
		config.Log.Print(err)
		return nil
	}

	if config.Configcli == nil {
		return errors.New("configcli is nil")
	}

	if len(privateZones) == 0 {
		// fix up any clusters that we already upgraded
		if err := UpdateDNSes(ctx, config.Configcli.ConfigV1().DNSes(), config.ResourceGroupID); err != nil {
			config.Log.Print(err)
		}
		return nil
	}

	if config.Mcocli == nil {
		return errors.New("mcocli is nil")
	}

	if config.Kubernetescli == nil {
		return errors.New("kubernetescli is nil")
	}

	mcpLister := config.Mcocli.MachineconfigurationV1().MachineConfigPools()
	nodeLister := config.Kubernetescli.CoreV1().Nodes()
	sameNumberOfNodesAndMachines, err := ready.SameNumberOfNodesAndMachines(ctx, mcpLister, nodeLister)
	if err != nil {
		config.Log.Print(err)
		return nil
	}

	if !sameNumberOfNodesAndMachines {
		return nil
	}

	clusterVersionIsLessThan4_4, err := version.ClusterVersionIsLessThan4_4(ctx, config.Configcli)
	if err != nil {
		config.Log.Print(err)
		return nil
	}

	if clusterVersionIsLessThan4_4 {
		config.Log.Printf("cluster version < 4.4, not removing private DNS zone")
		return nil
	}

	if err = UpdateDNSes(ctx, config.Configcli.ConfigV1().DNSes(), config.ResourceGroupID); err != nil {
		config.Log.Print(err)
		return nil
	}

	if err := RemoveZones(ctx, config.VNetLinksClient, config.PrivateZonesClient, privateZones, resourceGroup); err != nil {
		config.Log.Print(err)
		return nil
	}
	return nil
}

func DeletePrivateDNSVNetLinks(ctx context.Context, vNetLinksClient privatedns.VirtualNetworkLinksClient, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	if vNetLinksClient == nil {
		return errors.New("vNetLinksClient is nil")
	}

	vNetLinks, err := vNetLinksClient.List(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}

	for _, vNetLink := range vNetLinks {
		err = vNetLinksClient.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, *vNetLink.Name, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func RemoveZones(ctx context.Context,
	vNetLinksClient privatedns.VirtualNetworkLinksClient,
	privateZoneClient privatedns.PrivateZonesClient,
	privateZones []mgmtprivatedns.PrivateZone,
	resourceGroup string) error {
	for _, privateZone := range privateZones {
		if err := DeletePrivateDNSVNetLinks(ctx, vNetLinksClient, *privateZone.ID); err != nil {
			return err
		}

		r, err := azure.ParseResourceID(*privateZone.ID)
		if err != nil {
			return err
		}

		if privateZoneClient == nil {
			return errors.New("privateZoneClient is nil")
		}

		if err := privateZoneClient.DeleteAndWait(ctx, resourceGroup, r.ResourceName, ""); err != nil {
			return err
		}
	}
	return nil
}

func UpdateDNSes(ctx context.Context, dnsI DNSIClient, resourceGroupID string) error {
	fn := updateClusterDNSFn(ctx, dnsI, resourceGroupID)
	return retry.RetryOnConflict(retry.DefaultRetry, fn)
}

type DNSIClient interface {
	v1.DNSInterface
}

func updateClusterDNSFn(ctx context.Context, dnsClient DNSIClient, resourceGroupID string) func() error {
	return func() error {
		if dnsClient == nil {
			return errors.New("dnsClient interface is nil")
		}

		dns, err := dnsClient.Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if dns.Spec.PrivateZone == nil ||
			!strings.HasPrefix(
				strings.ToLower(dns.Spec.PrivateZone.ID),
				strings.ToLower(resourceGroupID)) {
			return nil
		}

		dns.Spec.PrivateZone = nil

		_, err = dnsClient.Update(ctx, dns, metav1.UpdateOptions{})
		return err
	}
}
