package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
)

// UnmarshalJSON unmarshals tags.  We override this to ensure that PATCH
// behaviour overwrites an existing tags map rather than endlessly adding to it
func (t *Tags) UnmarshalJSON(b []byte) error {
	var m map[string]string
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	*t = m
	return nil
}
