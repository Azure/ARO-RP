package env

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/env/dev"
	"github.com/jim-minter/rp/pkg/env/prod"
	"github.com/jim-minter/rp/pkg/env/shared/dns"
)

type Interface interface {
	CosmosDB(ctx context.Context) (string, string, error)
	DNS() dns.Manager
	FPAuthorizer(ctx context.Context) (autorest.Authorizer, error)
	IsReady() bool
	ListenTLS(ctx context.Context) (net.Listener, error)
	Authenticated(h http.Handler) http.Handler
	Location() string
	ResourceGroup() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return dev.New(ctx, log)
	}
	return prod.New(ctx, log)
}
