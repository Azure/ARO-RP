package fake

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type fakeGenerator struct {
	words      []string
	currentPos int
}

func NewGenerator(predefinedWords []string) uuid.Generator {
	return &fakeGenerator{
		words: predefinedWords,
	}
}

func (f *fakeGenerator) movePos() {
	f.currentPos++
}

func (f fakeGenerator) Generate() string {
	defer f.movePos()
	if len(f.words) < f.currentPos {
		return ""
	}
	return f.words[f.currentPos]
}
