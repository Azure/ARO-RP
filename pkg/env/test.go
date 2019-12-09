package env

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"

	"github.com/Azure/go-autorest/autorest"

	"github.com/jim-minter/rp/pkg/util/clientauthorizer"
)

type test struct {
	*prod

	l net.Listener

	TLSKey   *rsa.PrivateKey
	TLSCerts []*x509.Certificate
}

func NewTest(l net.Listener, cert []byte) *test {
	return &test{
		prod: &prod{
			ClientAuthorizer: clientauthorizer.NewOne(cert),
		},
		l: l,
	}
}

func (t *test) FPAuthorizer(ctx context.Context, resource string) (autorest.Authorizer, error) {
	return nil, nil
}

func (t *test) GetSecret(ctx context.Context, secretName string) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	switch secretName {
	case "tls":
		return t.TLSKey, t.TLSCerts, nil
	default:
		return nil, nil, fmt.Errorf("secret %q not found", secretName)
	}
}

func (t *test) Listen() (net.Listener, error) {
	return t.l, nil
}
