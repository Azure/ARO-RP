package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type FakeACRToken struct {
	now                func() time.Time
	generatedPasswords []string
}

var _ acrtoken.Manager = &FakeACRToken{}

func New(now func() time.Time) *FakeACRToken {
	return &FakeACRToken{
		now:                now,
		generatedPasswords: make([]string, 0),
	}
}

func (f *FakeACRToken) GetGeneratedPasswords() []string {
	return f.generatedPasswords
}

// Delete implements [acrtoken.Manager].
func (f *FakeACRToken) Delete(ctx context.Context, registryProfile *api.RegistryProfile) error {
	panic("unimplemented")
}

// EnsureTokenAndPassword implements [acrtoken.Manager].
func (f *FakeACRToken) EnsureTokenAndPassword(ctx context.Context, registryProfile *api.RegistryProfile) (string, error) {
	panic("unimplemented")
}

// NewRegistryProfile implements [acrtoken.Manager].
func (f *FakeACRToken) NewRegistryProfile() *api.RegistryProfile {
	panic("unimplemented")
}

// RotateTokenPassword implements [acrtoken.Manager].
func (f *FakeACRToken) RotateTokenPassword(ctx context.Context, registryProfile *api.RegistryProfile) error {
	newTestPassword := uuid.DefaultGenerator.Generate()
	f.generatedPasswords = append(f.generatedPasswords, newTestPassword)
	registryProfile.Password = api.SecureString(newTestPassword)
	registryProfile.IssueDate = pointerutils.ToPtr(f.now())
	return nil
}
