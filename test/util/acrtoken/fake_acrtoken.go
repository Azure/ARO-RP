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
	acrDomain          string
	now                func() time.Time
	generatedPasswords []string
}

var _ acrtoken.Manager = &FakeACRToken{}

func New(now func() time.Time, acrDomain string) *FakeACRToken {
	return &FakeACRToken{
		now:                now,
		acrDomain:          acrDomain,
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

func (f *FakeACRToken) generateTokenPassword(registryProfile *api.RegistryProfile) string {
	newTestPassword := uuid.DefaultGenerator.Generate()
	f.generatedPasswords = append(f.generatedPasswords, newTestPassword)
	registryProfile.Password = api.SecureString(newTestPassword)
	registryProfile.IssueDate = pointerutils.ToPtr(f.now().UTC())
	return newTestPassword
}

// EnsureTokenAndPassword implements [acrtoken.Manager].
func (f *FakeACRToken) EnsureTokenAndPassword(ctx context.Context, registryProfile *api.RegistryProfile) (string, error) {
	return f.generateTokenPassword(registryProfile), nil
}

// NewRegistryProfile implements [acrtoken.Manager].
func (f *FakeACRToken) NewRegistryProfile() *api.RegistryProfile {
	currentTime := f.now().UTC()
	return &api.RegistryProfile{
		Name:      f.acrDomain,
		Username:  "testuser",
		IssueDate: &currentTime,
	}
}

// RotateTokenPassword implements [acrtoken.Manager].
func (f *FakeACRToken) RotateTokenPassword(ctx context.Context, registryProfile *api.RegistryProfile) error {
	_ = f.generateTokenPassword(registryProfile)
	return nil
}
