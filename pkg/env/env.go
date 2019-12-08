package env

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/env/prod"
	"github.com/jim-minter/rp/pkg/env/shared/dns"
)

type Interface interface {
	CosmosDB(context.Context) (string, string, error)
	DNS() dns.Manager
	FPAuthorizer(context.Context, string) (autorest.Authorizer, error)
	GetSecret(context.Context, string) (*rsa.PrivateKey, []*x509.Certificate, error)
	IsReady() bool
	Listen() (net.Listener, error)
	Authenticated(http.Handler) http.Handler
	Location() string
	ResourceGroup() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return newDev(ctx, log)
	}
	return prod.New(ctx, log)
}
