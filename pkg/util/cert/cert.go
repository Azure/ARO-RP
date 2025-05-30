package cert

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha1"
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

func Thumbprint(cert *x509.Certificate) string {
	return fmt.Sprintf("%X", sha1.Sum(cert.Raw))
}
