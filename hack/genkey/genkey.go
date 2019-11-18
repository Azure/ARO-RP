package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

func run(name string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	now := time.Now()

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now,
		NotAfter:              now.AddDate(1, 0, 0),
		Subject:               pkix.Name{CommonName: name},
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	// key in der format
	err = ioutil.WriteFile(name+".key", x509.MarshalPKCS1PrivateKey(key), 0600)
	if err != nil {
		return err
	}

	// cert in der format
	err = ioutil.WriteFile(name+".crt", cert, 0666)
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

	err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if err != nil {
		return err
	}

	// key and cert in PKCS#8 PEM format for Azure Key Vault.
	return ioutil.WriteFile(name+".pem", buf.Bytes(), 0600)
}

func main() {
	if err := run(os.Args[1]); err != nil {
		panic(err)
	}
}
