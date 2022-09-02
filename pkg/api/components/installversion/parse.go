package installversion

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
)

func FromExternalBytes(body *[]byte) (*openShiftCluster, error) {
	r := &openShiftCluster{}

	err := json.Unmarshal(*body, &r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
