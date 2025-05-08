package pem

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

var _ = Describe("PEM", func() {
	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	Expect(err).ToNot(HaveOccurred())

	Describe("encoding keys", func() {
		It("succeeds", func() {
			keyOut, err := Encode(validCaKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(keyOut).To(ContainSubstring("BEGIN RSA PRIVATE KEY"))
		})
	})

	Describe("encoding single certificate", func() {
		It("succeeds", func() {
			certsOut, err := Encode(validCaCerts...)
			Expect(err).ToNot(HaveOccurred())
			Expect(certsOut).To(ContainSubstring("BEGIN CERTIFICATE"))
		})
	})

	Describe("encoding multiple certificates", func() {
		It("succeeds", func() {
			certsOut, err := Encode(validCaCerts[0], validCaCerts[0])
			Expect(err).ToNot(HaveOccurred())
			Expect(bytes.Count(certsOut, []byte("BEGIN CERTIFICATE"))).To(Equal(2))
		})
	})
})

func TestPEM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PEM Suite")
}
