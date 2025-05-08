package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"

	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

func run(name string, flags flagsType) error {
	var signingKey *rsa.PrivateKey
	var signingCert *x509.Certificate

	if *flags.keyFile != "" {
		b, err := os.ReadFile(*flags.keyFile)
		if err != nil {
			return err
		}

		signingKey, err = x509.ParsePKCS1PrivateKey(b)
		if err != nil {
			return err
		}
	}

	if *flags.certFile != "" {
		b, err := os.ReadFile(*flags.certFile)
		if err != nil {
			return err
		}

		signingCert, err = x509.ParseCertificate(b)
		if err != nil {
			return err
		}
	}

	key, cert, err := utiltls.GenerateKeyAndCertificate(name, signingKey, signingCert, *flags.ca, *flags.client)
	if err != nil {
		return err
	}

	// key in der format
	err = os.WriteFile(name+".key", x509.MarshalPKCS1PrivateKey(key), 0600)
	if err != nil {
		return err
	}

	// cert in der format
	err = os.WriteFile(name+".crt", cert[0].Raw, 0666)
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
	return os.WriteFile(name+".pem", buf.Bytes(), 0600)
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "usage: %s commonName\n", os.Args[0])
	flag.PrintDefaults()
}

type flagsType struct {
	client   *bool
	ca       *bool
	keyFile  *string
	certFile *string
}

func main() {
	flags := flagsType{
		client:   flag.Bool("client", false, "generate client certificate"),
		ca:       flag.Bool("ca", false, "generate ca certificate"),
		keyFile:  flag.String("keyFile", "", `file containing signing key in der format (default "" - self-signed)`),
		certFile: flag.String("certFile", "", `file containing signing certificate in der format (default "" - self-signed)`),
	}

	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(flag.Arg(0), flags); err != nil {
		panic(err)
	}
}
