package env

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"net"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/util/clientauthorizer"
	"github.com/jim-minter/rp/pkg/util/dns"
	"github.com/jim-minter/rp/pkg/util/instancemetadata"
)

type Interface interface {
	clientauthorizer.ClientAuthorizer
	instancemetadata.InstanceMetadata

	CosmosDB(context.Context) (string, string)
	DNS() dns.Manager
	FPAuthorizer(context.Context, string) (autorest.Authorizer, error)
	GetSecret(context.Context, string) (*rsa.PrivateKey, []*x509.Certificate, error)
	Listen() (net.Listener, error)
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return newDev(ctx, log)
	}

	im, err := instancemetadata.NewProd()
	if err != nil {
		return nil, err
	}

	return newProd(ctx, log, im, clientauthorizer.NewARM(log))
}
