package encrypt

import (
	"testing"
)

func TestCryptorRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	chacha, err := New(key)
	if err != nil {
		t.Error(err)
	}

	test := "topSecret"
	encrypted, err := chacha.Encrypt(test)
	if err != nil {
		t.Error(err)
	}

	decrypted, err := chacha.Decrypt(encrypted)
	if err != nil {
		t.Error(err)
	}

	if test != decrypted {
		t.Error("encryption roundTrip failed")
	}

}
