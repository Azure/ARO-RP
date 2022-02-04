package tls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/templates/content/bootkube"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	baremetaltypes "github.com/openshift/installer/pkg/types/baremetal"
	openstacktypes "github.com/openshift/installer/pkg/types/openstack"
	ovirttypes "github.com/openshift/installer/pkg/types/ovirt"
	vspheretypes "github.com/openshift/installer/pkg/types/vsphere"
)

// MCSCertKey is the asset that generates the MCS key/cert pair.
type MCSCertKey struct {
	SignedCertKey
}

var _ asset.Asset = (*MCSCertKey)(nil)

// Dependencies returns the dependency of the the cert/key pair, which includes
// the parent CA, and install config if it depends on the install config for
// DNS names, etc.
func (a *MCSCertKey) Dependencies() []asset.Asset {
	return []asset.Asset{
		&RootCA{},
		&installconfig.InstallConfig{},
		&bootkube.ARODNSConfig{},
	}
}

// Generate generates the cert/key pair based on its dependencies.
func (a *MCSCertKey) Generate(dependencies asset.Parents) error {
	ca := &RootCA{}
	installConfig := &installconfig.InstallConfig{}
	aroDNSConfig := &bootkube.ARODNSConfig{}
	dependencies.Get(ca, installConfig, aroDNSConfig)

	hostname := internalAPIAddress(installConfig.Config)

	cfg := &CertCfg{
		Subject:      pkix.Name{CommonName: "system:machine-config-server"},
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		Validity:     ValidityTenYears,
	}

	switch installConfig.Config.Platform.Name() {
	case azuretypes.Name:
		if installConfig.Config.Azure.IsARO() {
			cfg.IPAddresses = []net.IP{net.ParseIP(aroDNSConfig.APIIntIP)}
			cfg.DNSNames = []string{hostname, aroDNSConfig.APIIntIP}
		} else {
			cfg.DNSNames = []string{hostname}
		}
	case baremetaltypes.Name:
		cfg.IPAddresses = []net.IP{net.ParseIP(installConfig.Config.BareMetal.APIVIP)}
		cfg.DNSNames = []string{hostname, installConfig.Config.BareMetal.APIVIP}
	case openstacktypes.Name:
		cfg.IPAddresses = []net.IP{net.ParseIP(installConfig.Config.OpenStack.APIVIP)}
		cfg.DNSNames = []string{hostname, installConfig.Config.OpenStack.APIVIP}
	case ovirttypes.Name:
		cfg.IPAddresses = []net.IP{net.ParseIP(installConfig.Config.Ovirt.APIVIP)}
		cfg.DNSNames = []string{hostname, installConfig.Config.Ovirt.APIVIP}
	case vspheretypes.Name:
		cfg.DNSNames = []string{hostname}
		if installConfig.Config.VSphere.APIVIP != "" {
			cfg.IPAddresses = []net.IP{net.ParseIP(installConfig.Config.VSphere.APIVIP)}
			cfg.DNSNames = append(cfg.DNSNames, installConfig.Config.VSphere.APIVIP)
		}
	default:
		cfg.DNSNames = []string{hostname}
	}

	return a.SignedCertKey.Generate(cfg, ca, "machine-config-server", DoNotAppendParent)
}

// Name returns the human-friendly name of the asset.
func (a *MCSCertKey) Name() string {
	return "Certificate (mcs)"
}
