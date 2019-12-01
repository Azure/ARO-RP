package prod

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/env/shared"
)

type prod struct {
	*shared.Shared
	ms            *metadataService
	location      string
	resourceGroup string
}

func New(ctx context.Context, log *logrus.Entry) (*prod, error) {
	location, subscriptionID, resourceGroup, err := getMetadata()
	if err != nil {
		return nil, err
	}

	p := &prod{
		ms:            NewMetadataService(log),
		location:      location,
		resourceGroup: resourceGroup,
	}

	p.Shared, err = shared.NewShared(ctx, log, subscriptionID, resourceGroup)
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
		MinVersion: tls.VersionTLS12,
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

func (p *prod) Location() string {
	return p.location
}

func (p *prod) ResourceGroup() string {
	return p.resourceGroup
}

func getMetadata() (string, string, string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/metadata/instance/compute?api-version=2019-03-11", nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Metadata", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("unexpected status code %q", resp.StatusCode)
	}

	if strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0] != "application/json" {
		return "", "", "", fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	var m *struct {
		Location          string `json:"location,omitempty"`
		ResourceGroupName string `json:"resourceGroupName,omitempty"`
		SubscriptionID    string `json:"subscriptionId,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return "", "", "", err
	}

	return m.Location, m.SubscriptionID, m.ResourceGroupName, nil
}
