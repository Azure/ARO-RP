package prod

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/env/shared"
)

type prod struct {
	*shared.Shared
	ms *metadataService
}

func New(ctx context.Context, log *logrus.Entry, subscriptionId, resourceGroup string) (*prod, error) {
	var err error

	p := &prod{
		ms: NewMetadataService(log),
	}

	p.Shared, err = shared.NewShared(ctx, log, subscriptionId, resourceGroup)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prod) ListenTLS(ctx context.Context) (net.Listener, error) {
	key, cert, err := p.GetSecret(ctx, "tls")
	if err != nil {
		return nil, err
	}

	return tls.Listen("tcp", ":8443", &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					cert.Raw,
				},
				PrivateKey: key,
			},
		},
		ClientAuth: tls.RequestClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 || !p.ms.allowClientCertificate(rawCerts[0]) {
				return errors.New("invalid certificate")
			}
			return nil
		},
	})
}

func (p *prod) IsReady() bool {
	return p.ms.isReady()
}
