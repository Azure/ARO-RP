// Package etm provides a set of Encrypt-Then-MAC AEAD implementations, which
// combine block ciphers like AES with HMACs.
//
// AEADs
//
// An AEAD (Authenticated Encryption with Associated Data) construction provides
// a unified API for sealing messages in a way which provides both
// confidentiality *and* integrity.
//
// This not only prevents malicious tampering but also eliminates online attacks
// like padding oracle attacks which can allow an attacker to recover plaintexts
// without knowledge of the secret key (e.g., Lucky 13 attack, BEAST attack,
// etc.).
//
// By rejecting ciphertexts which have been modified, these types of attacks are
// eliminated.
//
// Constructions
//
// This package implements five proposed standards:
//
//		AEAD_AES_128_CBC_HMAC_SHA_256
//		AEAD_AES_192_CBC_HMAC_SHA_384
//		AEAD_AES_256_CBC_HMAC_SHA_384
//		AEAD_AES_256_CBC_HMAC_SHA_512
//		AEAD_AES_128_CBC_HMAC_SHA1
//
// All five constructions combine AES in CBC mode with an HMAC, but vary in the
// degree of security offered and the amount of overhead required. See
// http://tools.ietf.org/html/draft-mcgrew-aead-aes-cbc-hmac-sha2-02 for full
// technical details.
//
// AES-128-CBC-HMAC-SHA-256
//
// AEAD_AES_128_CBC_HMAC_SHA_256 requires a 32-byte key, provides 128 bits of
// security for both confidentiality and integrity, and adds up to 56 bytes of
// overhead per message.
//
// AES-192-CBC-HMAC-SHA-384
//
// AEAD_AES_192_CBC_HMAC_SHA_384 requires a 48-byte key, provides 192 bits of
// security for both confidentiality and integrity, and adds up to 64 bytes of
// overhead per message.
//
// AES-256-CBC-HMAC-SHA-384
//
// AEAD_AES_256_CBC_HMAC_SHA_384 requires a 56-byte key, provides 256 bits of
// security for confidentiality, provides 192 bits of security for integrity, and
// adds up to 64 bytes of overhead per message.
//
// AES-256-CBC-HMAC-SHA-512
//
// AEAD_AES_256_CBC_HMAC_SHA_512 requires a 64-byte key, provides 256 bits of
// security for both confidentiality and integrity, and adds up to 72 bytes of
// overhead per message.
//
// AES-128-CBC-HMAC-SHA1
//
// AEAD_AES_128_CBC_HMAC_SHA1 requires a 36-byte key, provides 128 bits of
// security for confidentiality, provides 96 bits of security for integrity, and
// adds up to 52 bytes of overhead per message. (This construction uses SHA-1,
// and should only be used when the use of SHA2 is not possible.)
package etm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
)

// NewAES128SHA256 returns an AEAD_AES_128_CBC_HMAC_SHA_256 instance given a
// 32-byte key or an error if the key is the wrong size.
// AEAD_AES_128_CBC_HMAC_SHA_256 combines AES-128 in CBC mode with
// HMAC-SHA-256-128.
func NewAES128SHA256(key []byte) (cipher.AEAD, error) {
	return create(etmParams{
		cipherParams: aesCBC,
		macAlg:       sha256.New,
		encKeySize:   16,
		macKeySize:   16,
		tagSize:      16,
		key:          key,
	})
}

// NewAES192SHA384 returns an AEAD_AES_192_CBC_HMAC_SHA_384 instance given a
// 48-byte key or an error if the key is the wrong size.
// AEAD_AES_192_CBC_HMAC_SHA_384 combines AES-192 in CBC mode with
// HMAC-SHA-384-192.
func NewAES192SHA384(key []byte) (cipher.AEAD, error) {
	return create(etmParams{
		cipherParams: aesCBC,
		macAlg:       sha512.New384,
		encKeySize:   24,
		macKeySize:   24,
		tagSize:      24,
		key:          key,
	})
}

// NewAES256SHA384 returns an AEAD_AES_256_CBC_HMAC_SHA_384 instance given a
// 56-byte key or an error if the key is the wrong size.
// AEAD_AES_256_CBC_HMAC_SHA_384 combines AES-256 in CBC mode with
// HMAC-SHA-384-192.
func NewAES256SHA384(key []byte) (cipher.AEAD, error) {
	return create(etmParams{
		cipherParams: aesCBC,
		macAlg:       sha512.New384,
		encKeySize:   32,
		macKeySize:   24,
		tagSize:      24,
		key:          key,
	})
}

// NewAES256SHA512 returns an AEAD_AES_256_CBC_HMAC_SHA_512 instance given a
// 64-byte key or an error if the key is the wrong size.
// AEAD_AES_256_CBC_HMAC_SHA_512 combines AES-256 in CBC mode with
// HMAC-SHA-512-256.
func NewAES256SHA512(key []byte) (cipher.AEAD, error) {
	return create(etmParams{
		cipherParams: aesCBC,
		macAlg:       sha512.New,
		encKeySize:   32,
		macKeySize:   32,
		tagSize:      32,
		key:          key,
	})
}

// NewAES128SHA1 returns an AEAD_AES_128_CBC_HMAC_SHA1 instance given a 36-byte
// key or an error if the key is the wrong size.
// AEAD_AES_128_CBC_HMAC_SHA1 combines AES-128 in CBC mode with HMAC-SHA1-96.
func NewAES128SHA1(key []byte) (cipher.AEAD, error) {
	return create(etmParams{
		cipherParams: aesCBC,
		macAlg:       sha1.New,
		encKeySize:   16,
		macKeySize:   20,
		tagSize:      12,
		key:          key,
	})
}

type etmParams struct {
	cipherParams
	encKeySize, macKeySize, tagSize int

	key    []byte
	macAlg func() hash.Hash
}

func create(p etmParams) (cipher.AEAD, error) {
	l := p.encKeySize + p.macKeySize
	if len(p.key) != l {
		return nil, fmt.Errorf("etm: key must be %d bytes long", l)
	}
	encKey, macKey := split(p.key, p.encKeySize, p.macKeySize)
	return &etmAEAD{
		etmParams: p,
		encKey:    encKey,
		macKey:    macKey,
	}, nil
}

const (
	dataLenSize = 8
)

type etmAEAD struct {
	etmParams
	encKey, macKey []byte
}

func (aead *etmAEAD) Overhead() int {
	return aead.padSize + aead.tagSize + dataLenSize + aead.NonceSize()
}

func (aead *etmAEAD) NonceSize() int {
	return aead.nonceSize
}

func (aead *etmAEAD) Seal(dst, nonce, plaintext, data []byte) []byte {
	b, _ := aead.encAlg(aead.encKey) // guaranteed to work

	c := aead.encrypter(b, nonce)
	i := aead.pad(plaintext, aead.blockSize)
	s := make([]byte, len(i))
	c.CryptBlocks(s, i)
	s = append(nonce, s...)

	t := tag(hmac.New(aead.macAlg, aead.macKey), data, s, aead.tagSize)

	return append(dst, append(s, t...)...)
}

func (aead *etmAEAD) Open(dst, nonce, ciphertext, data []byte) ([]byte, error) {
	s := ciphertext[:len(ciphertext)-aead.tagSize]
	t := ciphertext[len(ciphertext)-aead.tagSize:]
	t2 := tag(hmac.New(aead.macAlg, aead.macKey), data, s, aead.tagSize)
	if nonce == nil {
		nonce = s[:aead.NonceSize()]
	}

	if !hmac.Equal(t, t2) {
		return nil, errors.New("message authentication failed")
	}

	b, _ := aead.encAlg(aead.encKey) // guaranteed to work

	c := aead.decrypter(b, nonce)
	o := make([]byte, len(s)-len(nonce))
	c.CryptBlocks(o, s[len(nonce):])

	return append(dst, aead.unpad(o, aead.blockSize)...), nil
}

type cipherParams struct {
	nonceSize, blockSize, padSize int

	encAlg    func(key []byte) (cipher.Block, error)
	encrypter func(cipher.Block, []byte) cipher.BlockMode
	decrypter func(cipher.Block, []byte) cipher.BlockMode
	pad       func([]byte, int) []byte
	unpad     func([]byte, int) []byte
}

// AES-CBC-PKCS7
var aesCBC = cipherParams{
	encAlg:    aes.NewCipher,
	blockSize: aes.BlockSize,
	nonceSize: aes.BlockSize,
	encrypter: cipher.NewCBCEncrypter,
	decrypter: cipher.NewCBCDecrypter,
	padSize:   aes.BlockSize,
	pad:       pkcs7pad,
	unpad:     pkcs7unpad,
}

func tag(h hash.Hash, data, s []byte, l int) []byte {
	al := make([]byte, dataLenSize)
	binary.BigEndian.PutUint64(al, uint64(len(data)*8)) // in bits
	h.Write(data)
	h.Write(s)
	h.Write(al)
	return h.Sum(nil)[:l]
}

func split(key []byte, encKeyLen, macKeyLen int) ([]byte, []byte) {
	return key[0:encKeyLen], key[len(key)-macKeyLen:]
}

func pkcs7pad(b []byte, blockSize int) []byte {
	ps := make([]byte, blockSize-(len(b)%blockSize))
	for i := range ps {
		ps[i] = byte(len(ps))
	}
	return append(b, ps...)
}

func pkcs7unpad(b []byte, blockSize int) []byte {
	return b[:len(b)-int(b[len(b)-1])]
}
