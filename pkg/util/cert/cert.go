package cert

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"time"
)

const DefaultMinDurationPercent = 0.20

// IsLessThanMinimumDuration indicates whether the provided cert has less
// than the provided minimum percentage of its duration remaining.
func IsLessThanMinimumDuration(cert *x509.Certificate, minDurationPercent float64) bool {
	duration := cert.NotAfter.Sub(cert.NotBefore)
	minDuration := time.Duration(float64(duration.Nanoseconds()) * DefaultMinDurationPercent)
	return time.Now().After(cert.NotAfter.Add(-minDuration))
}

func IsCertExpired(cert *x509.Certificate) bool {
	return time.Now().After(cert.NotAfter)
}

func DaysUntilExpiration(cert *x509.Certificate) int {
	return int(time.Until(cert.NotAfter) / (24 * time.Hour))
}
