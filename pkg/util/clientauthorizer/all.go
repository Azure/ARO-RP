package clientauthorizer

import (
	"crypto/tls"
)

type all struct{}

func NewAll() ClientAuthorizer {
	return &all{}
}

func (all) IsAuthorized(*tls.ConnectionState) bool {
	return true
}

func (all) IsReady() bool {
	return true
}
