package env

import (
	"context"
	"net"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

type prod struct {
	*shared
	ms            *armMetadataService
	location      string
	resourceGroup string
}

func newProd(ctx context.Context, log *logrus.Entry) (*prod, error) {
	tenantID, err := getTenantID()
	if err != nil {
		return nil, err
	}

	subscriptionID, location, resourceGroup, err := getInstanceMetadata()
	if err != nil {
		return nil, err
	}

	p := &prod{
		ms:            newARMMetadataService(log),
		location:      location,
		resourceGroup: resourceGroup,
	}

	p.shared, err = newShared(ctx, log, tenantID, subscriptionID, resourceGroup)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prod) Listen() (net.Listener, error) {
	return net.Listen("tcp", ":8443")
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

func (p *prod) FPAuthorizer(ctx context.Context, resource string) (autorest.Authorizer, error) {
	sp, err := p.fpToken(ctx, resource)
	if err != nil {
		return nil, err
	}

	return autorest.NewBearerAuthorizer(sp), nil
}

func (p *prod) IsReady() bool {
	return p.ms.isReady()
}

func (p *prod) Location() string {
	return p.location
}

func (p *prod) ResourceGroup() string {
	return p.resourceGroup
}
