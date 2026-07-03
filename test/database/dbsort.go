package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"cmp"

	"github.com/Azure/ARO-RP/pkg/database"
)

func CompareKeyable(a, b database.Keyable) int {
	return cmp.Compare(a.GetKey(), b.GetKey())
}
