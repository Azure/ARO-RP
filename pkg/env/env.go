package env

import (
	"context"
	"net"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/env/dev"
	"github.com/jim-minter/rp/pkg/env/prod"
)

type Interface interface {
	CosmosDB(ctx context.Context) (string, string, error)
	DNS(ctx context.Context) (string, error)
	FirstPartyAuthorizer(ctx context.Context) (autorest.Authorizer, error)
	IsReady() bool
	ListenTLS(ctx context.Context) (net.Listener, error)
}

func NewEnv(ctx context.Context, log *logrus.Entry, subscriptionId, resourceGroup string) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return dev.New(ctx, log, subscriptionId, resourceGroup)
	}
	return prod.New(ctx, log, subscriptionId, resourceGroup)
}
