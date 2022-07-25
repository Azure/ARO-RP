package interfaces

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "reflect"

func IsNil(v interface{}) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}
