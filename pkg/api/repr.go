// +build test

// stringifying representations of API documents for debugging and testing
// logging

package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"github.com/ugorji/go/codec"
)

func encodeJSON(i interface{}) string {
	w := &strings.Builder{}
	enc := codec.NewEncoder(w, &codec.JsonHandle{})
	err := enc.Encode(i)
	if err != nil {
		return err.Error()
	}
	return w.String()
}

func (c *SubscriptionDocument) String() string {
	return encodeJSON(c)
}

func (c *OpenShiftClusterDocument) String() string {
	return encodeJSON(c)
}

func (c *BillingDocument) String() string {
	return encodeJSON(c)
}
