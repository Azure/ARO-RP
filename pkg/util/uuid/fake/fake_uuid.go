package fake

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"sync"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type fakeGenerator struct {
	words      []string
	currentPos int
	mu         *sync.Mutex
}

func NewGenerator(predefinedWords []string) uuid.Generator {
	return &fakeGenerator{
		words: predefinedWords,
		mu:    &sync.Mutex{},
	}
}

func (f *fakeGenerator) movePos() {
	f.currentPos++
}

func (f *fakeGenerator) Generate() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	defer f.movePos()
	if len(f.words) < f.currentPos {
		return ""
	}
	return f.words[f.currentPos]
}
