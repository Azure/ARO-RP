package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"testing"
	"time"
)

// generateTestCert builds a small self-signed ECDSA certificate for tests.
// This avoids embedding PEM data and lets tests modify NotAfter deterministically.
func generateTestCert(t *testing.T) *x509.Certificate {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-cert"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(48 * time.Hour), // valid for 2 days
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse generated certificate: %v", err)
	}

	return cert
}

func TestIsCertExpiredAndDaysUntilExpiration(t *testing.T) {
	cert := generateTestCert(t)

	if IsCertExpired(cert) {
		t.Fatalf("expected future cert to be not expired")
	}

	days := DaysUntilExpiration(cert)
	if days < 1 || days > 3 {
		t.Fatalf("unexpected days until expiration: %d", days)
	}

	past := *cert
	past.NotAfter = time.Now().Add(-1 * time.Hour)
	if !IsCertExpired(&past) {
		t.Fatalf("expected past cert to be expired")
	}
}

func TestThumbprintSHA256(t *testing.T) {
	cert := generateTestCert(t)
	// Compute expected SHA-256 thumbprint using standard library
	expected := sha256.Sum256(cert.Raw)

	got := Thumbprint(cert)
	// Thumbprint returns uppercase hex without separators
	expectedHex := hexEncodeUpper(expected[:])

	if got != expectedHex {
		t.Fatalf("thumbprint mismatch: got %s expected %s", got, expectedHex)
	}
}

// hexEncodeUpper returns uppercase hex encoding of b without 0x prefix.
func hexEncodeUpper(b []byte) string {
	s := hex.EncodeToString(b)
	// encodeToString returns lowercase hex; convert to uppercase
	upper := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'f' {
			upper[i] = c - 'a' + 'A'
		} else {
			upper[i] = c
		}
	}
	return string(upper)
}

// TestLegacySHA1Example demonstrates how a client could compute a SHA-1
// thumbprint for backward compatibility. This ensures migration to SHA-256
// doesn't prevent clients from still computing SHA-1 on the certificate raw
// bytes if they require it.
func TestLegacySHA1Example(t *testing.T) {
	cert := generateTestCert(t)
	sum := sha1.Sum(cert.Raw)
	got := ThumbprintSHA1(cert)

	// convert to hex
	want := hexEncodeUpper(sum[:])
	if len(want) != 40 {
		t.Fatalf("unexpected hex length for sha1: %d", len(want))
	}
	if got != want {
		t.Fatalf("sha1 thumbprint mismatch: got %s expected %s", got, want)
	}
}
