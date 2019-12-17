package orderedmap

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// UnmarshalJSON unmarshals a JSON object into a slice of key/value structs,
// preserving key order.
func UnmarshalJSON(b []byte, i interface{}) error {
	xs := reflect.ValueOf(i).Elem()
	d := json.NewDecoder(bytes.NewReader(b))

	tok, err := d.Token()
	if err != nil {
		return err
	}
	if tok != json.Delim('{') {
		return fmt.Errorf("unexpected token %v", tok)
	}

	indexes := map[string]int{}
	for {
		tok, err = d.Token()
		if err != nil {
			return err
		}
		if tok == json.Delim('}') {
			break
		}
		k, ok := tok.(string)
		if !ok {
			return fmt.Errorf("unexpected token %v", tok)
		}

		kv := reflect.New(xs.Type().Elem()).Elem()
		kv.Field(0).SetString(k)
		err = d.Decode(kv.Field(1).Addr().Interface())
		if err != nil {
			return err
		}

		if i, found := indexes[k]; found {
			xs.Index(i).Set(kv)
		} else {
			indexes[k] = xs.Len()
			xs = reflect.Append(xs, kv)
		}
	}

	reflect.ValueOf(i).Elem().Set(xs)

	return nil
}

// MarshalJSON unmarshals a slice of key/value structs into a JSON object,
// preserving key order.
func MarshalJSON(i interface{}) ([]byte, error) {
	if i == nil {
		return []byte("null"), nil
	}

	buf := &bytes.Buffer{}
	buf.WriteByte('{')

	xs := reflect.ValueOf(i)
	for i := 0; i < xs.Len(); i++ {
		b, err := json.Marshal(xs.Index(i).Field(0).String())
		if err != nil {
			return nil, err
		}
		buf.Write(b)
		buf.WriteByte(':')
		b, err = json.Marshal(xs.Index(i).Field(1).Interface())
		if err != nil {
			return nil, err
		}
		buf.Write(b)
		if i < xs.Len()-1 {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}
