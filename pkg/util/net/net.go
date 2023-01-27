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
	v1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
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

func managedDomainSuffixes() []string {
	return []string{
		publicCloudManagedDomainSuffix,
		govCloudManagedDomainSuffix,
	}
}

func IsManagedDomain(domain string) bool {
	suffixes := managedDomainSuffixes()
	for _, suffix := range suffixes {
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}
	return false
}

func DeletePrivateDNSVirtualNetworkLinks(ctx context.Context, virtualNetworkLinksClient privatedns.VirtualNetworkLinksClient, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	virtualNetworkLinks, err := virtualNetworkLinksClient.List(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}

	for _, virtualNetworkLink := range virtualNetworkLinks {
		err = virtualNetworkLinksClient.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, *virtualNetworkLink.Name, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func RemoveZones(ctx context.Context, log *logrus.Entry, virtualNetworkLinksClient privatedns.VirtualNetworkLinksClient, privateZonesClient privatedns.PrivateZonesClient, privateZones []mgmtprivatedns.PrivateZone, resourceGroup string) {
	for _, privateZone := range privateZones {
		if err := DeletePrivateDNSVirtualNetworkLinks(ctx, virtualNetworkLinksClient, *privateZone.ID); err != nil {
			log.Print(err)
			return
		}

		r, err := azure.ParseResourceID(*privateZone.ID)
		if err != nil {
			log.Print(err)
			return
		}

		if err = privateZonesClient.DeleteAndWait(ctx, resourceGroup, r.ResourceName, ""); err != nil {
			log.Print(err)
			return
		}
	}
}

func UpdateDNSs(ctx context.Context, dnsInterface v1.DNSInterface, resourceGroupID string) error {
	fn := updateClusterDNSFn(ctx, dnsInterface, resourceGroupID)
	return retry.RetryOnConflict(retry.DefaultRetry, fn)
}

func updateClusterDNSFn(ctx context.Context, dnsInterface v1.DNSInterface, resourceGroupID string) func() error {
	return func() error {
		dns, err := dnsInterface.Get(ctx, "cluster", metav1.GetOptions{})
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

		_, err = dnsInterface.Update(ctx, dns, metav1.UpdateOptions{})
		return err
	}
}

func McpContainsARODNSConfig(mcp mcv1.MachineConfigPool) bool {
	for _, source := range mcp.Status.Configuration.Source {
		mcpPrefix := "99-"
		mcpSuffix := "-aro-dns"

		if source.Name == mcpPrefix+mcp.Name+mcpSuffix {
			return true
		}
	}
	return false
}
