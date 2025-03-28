package uuid

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	gofrsuuid "github.com/gofrs/uuid"

	apiuuid "github.com/Azure/ARO-RP/pkg/api/util/uuid"
)

var DefaultGenerator apiuuid.Generator = apiuuid.DefaultGenerator

type Generator = apiuuid.Generator

func FromString(u string) (gofrsuuid.UUID, error) {
	return gofrsuuid.FromString(u)
}

func MustFromString(u string) gofrsuuid.UUID {
	return gofrsuuid.Must(gofrsuuid.FromString(u))
}

func IsValid(u string) bool {
	return apiuuid.IsValid(u)
}
