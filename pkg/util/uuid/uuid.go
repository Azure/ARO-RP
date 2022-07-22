package uuid

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	gofrsuuid "github.com/gofrs/uuid"
)

type Generator interface {
	Generate() string
}

type defaultGenerator struct{}

func (d defaultGenerator) Generate() string {
	return gofrsuuid.Must(gofrsuuid.DefaultGenerator.NewV4()).String()
}

var DefaultGenerator Generator = defaultGenerator{}

func FromString(u string) (gofrsuuid.UUID, error) {
	return gofrsuuid.FromString(u)
}

func MustFromString(u string) gofrsuuid.UUID {
	return gofrsuuid.Must(gofrsuuid.FromString(u))
}

func IsValid(u string) bool {
	_, err := gofrsuuid.FromString(u)
	return err == nil
}
