package prod

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
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
	})
}

func (p *prod) Authenticated(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil ||
			len(r.TLS.PeerCertificates) == 0 ||
			!p.ms.allowClientCertificate(r.TLS.PeerCertificates[0].Raw) {
			api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (p *prod) IsReady() bool {
	return p.ms.isReady()
}
