package env

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

type prod struct {
	*shared
	ms            *armMetadataService
	location      string
	resourceGroup string
}

type azureClaim struct {
	TenantID string `json:"tid,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
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

func (p *prod) IsReady() bool {
	return p.ms.isReady()
}

func (p *prod) Location() string {
	return p.location
}

func (p *prod) ResourceGroup() string {
	return p.resourceGroup
}

func getTenantID() (string, error) {
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		return "", err
	}

	token, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	err = token.EnsureFresh()
	if err != nil {
		return "", err
	}

	p := &jwt.Parser{}
	c := &azureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return "", err
	}

	return c.TenantID, nil
}

func getInstanceMetadata() (string, string, string, error) {
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

	return m.SubscriptionID, m.Location, m.ResourceGroupName, nil
}
