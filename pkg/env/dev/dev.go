package dev

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/env/shared"
)

type dev struct {
	*shared.Shared
}

func New(ctx context.Context, log *logrus.Entry, subscriptionId, resourceGroup string) (*dev, error) {
	var err error

	d := &dev{}

	d.Shared, err = shared.NewShared(ctx, log, subscriptionId, resourceGroup)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *dev) ListenTLS(ctx context.Context) (net.Listener, error) {
	key, cert, err := d.GetSecret(ctx, "tls")
	if err != nil {
		return nil, err
	}

	// no TLS client cert verification in dev mode, but we'll only listen on
	// localhost
	return tls.Listen("tcp", "localhost:8443", &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					cert.Raw,
				},
				PrivateKey: key,
			},
		},
	})
}

func (d *dev) IsReady() bool {
	return true
}
