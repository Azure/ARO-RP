package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// MissingFields retains values that do not map to struct fields during JSON
// marshalling/unmarshalling.  MissingFields implements
// github.com/ugorji/go/codec.MissingFielder.
type MissingFields struct {
	m map[string]interface{}
}

// CodecMissingField is called to set a missing field and value pair
func (mf *MissingFields) CodecMissingField(field []byte, value interface{}) bool {
	if mf.m == nil {
		mf.m = map[string]interface{}{}
	}
	(mf.m)[string(field)] = value
	return true
}

// CodecMissingFields returns the set of fields which are not struct fields
func (mf *MissingFields) CodecMissingFields() map[string]interface{} {
	return mf.m
}
