package clientauthorizer

import "crypto/tls"

type ClientAuthorizer interface {
	IsAuthorized(*tls.ConnectionState) bool
	IsReady() bool
}
