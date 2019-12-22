package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
)

type Version [3]byte

func newVersion(vsn string) (v Version, err error) {
	_, err = fmt.Sscanf(vsn, "%d.%d.%d", &v[0], &v[1], &v[2])
	return
}

func (v Version) Lt(w Version) bool {
	return v[0] < w[0] || v[1] < w[1] || v[2] < w[2]
}
