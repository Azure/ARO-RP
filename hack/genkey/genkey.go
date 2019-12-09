package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"strings"

	"github.com/jim-minter/rp/pkg/util/tls"
)

var (
	extKeyUsage = flag.String("extKeyUsage", "server", "server or client")
)

func run(name string) error {
	key, cert, err := tls.GenerateKeyAndCertificate(name, strings.EqualFold(*extKeyUsage, "client"))
	if err != nil {
		return err
	}

	// key in der format
	err = ioutil.WriteFile(name+".key", x509.MarshalPKCS1PrivateKey(key), 0600)
	if err != nil {
		return err
	}

	// cert in der format
	err = ioutil.WriteFile(name+".crt", cert[0].Raw, 0666)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	err = pem.Encode(buf, &pem.Block{Type: "PRIVATE KEY", Bytes: b})
	if err != nil {
		return err
	}

	err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert[0].Raw})
	if err != nil {
		return err
	}

	// key and cert in PKCS#8 PEM format for Azure Key Vault.
	return ioutil.WriteFile(name+".pem", buf.Bytes(), 0600)
}

func main() {
	flag.Parse()

	if err := run(flag.Arg(0)); err != nil {
		panic(err)
	}
}
