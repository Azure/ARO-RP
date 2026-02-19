package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"cmp"

	"github.com/Azure/ARO-RP/pkg/database"
)

func CompareIDable(a, b database.IDable) int {
	return cmp.Compare(a.GetID(), b.GetID())
}
