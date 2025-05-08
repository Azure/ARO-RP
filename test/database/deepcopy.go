package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"

	"github.com/ugorji/go/codec"
)

var h = &codec.JsonHandle{}

func deepCopy(i interface{}) (interface{}, error) {
	var b []byte
	err := codec.NewEncoderBytes(&b, h).Encode(i)
	if err != nil {
		return nil, err
	}

	i = reflect.New(reflect.ValueOf(i).Elem().Type()).Interface()
	err = codec.NewDecoderBytes(b, h).Decode(&i)
	if err != nil {
		return nil, err
	}

	return i, nil
}
