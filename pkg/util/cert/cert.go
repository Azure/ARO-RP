package cert

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"time"
)

func IsCertExpired(cert *x509.Certificate) bool {
	return time.Now().After(cert.NotAfter)
}

func DaysUntilExpiration(cert *x509.Certificate) int {
	return int(time.Until(cert.NotAfter) / (24 * time.Hour))
}

// Thumbprint returns the SHA-256 string thumbprint of the certificate in uppercase hex
func Thumbprint(cert *x509.Certificate) string {
	return fmt.Sprintf("%X", sha256.Sum256(cert.Raw))
}

// Obsolete: compatibility helper during migration to SHA-256
func ThumbprintSHA1(cert *x509.Certificate) string {
    return fmt.Sprintf("%X", sha1.Sum(cert.Raw))
}