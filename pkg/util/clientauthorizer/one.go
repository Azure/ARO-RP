package clientauthorizer

import (
	"bytes"
)

type one struct {
	cert []byte
}

func NewOne(cert []byte) ClientAuthorizer {
	return &one{
		cert: cert,
	}
}

func (o *one) IsAuthorized(b []byte) bool {
	return bytes.Equal(o.cert, b)
}

func (one) IsReady() bool {
	return true
}
