package encrypt

import (
	"crypto/rsa"
	"crypto/x509"
	"reflect"

	"github.com/ugorji/go/codec"
)

// AddExtensions adds extensions to a ugorji/go/codec to enable it to serialise
// our types properly. If cipher is provided, it will encrypt/decrypt sensitive fields
func AddExtensions(h *codec.BasicHandle, cipher Cipher) error {
	err := h.AddExt(reflect.TypeOf(&rsa.PrivateKey{}), 0, func(v reflect.Value) ([]byte, error) {
		if reflect.DeepEqual(v.Elem().Interface(), rsa.PrivateKey{}) {
			return nil, nil
		}
		data := x509.MarshalPKCS1PrivateKey(v.Interface().(*rsa.PrivateKey))

		if cipher != nil {
			return cipher.Encrypt(data)
		}

		return data, nil
	}, func(v reflect.Value, b []byte) error {
		var err error
		if cipher != nil {
			b, err = cipher.Decrypt(b)
		}

		key, err := x509.ParsePKCS1PrivateKey(b)
		if err != nil {
			return err
		}
		v.Elem().Set(reflect.ValueOf(key).Elem())
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
