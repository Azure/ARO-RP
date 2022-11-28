package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/go-test/deep"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestList(t *testing.T) {
	testListTags := func(ctx context.Context, sys *types.SystemContext, ref types.ImageReference) ([]string, error) {
		return []string{
			"abc", "def",
		}, nil
	}

	m := &repositoryMirrorManager{
		getRepositoryTags: testListTags,
	}

	nt, err := m.List(context.Background(), "quay.io/app-sre/managed-upgrade-operator:v0.1.856-eebbe07", nil)
	if err != nil {
		t.Fatal(err)
	}

	r := make([]string, 0, len(nt))

	for _, tag := range nt {
		r = append(r, tag.String())
	}

	expected := []string{
		"quay.io/app-sre/managed-upgrade-operator:abc",
		"quay.io/app-sre/managed-upgrade-operator:def",
	}

	for _, err := range deep.Equal(r, expected) {
		t.Error(err)
	}
}

func TestMirror(t *testing.T) {
	hook, log := testlog.New()
	sentRefs := make([][2]types.ImageReference, 0)
	dstAuth := &types.DockerAuthConfig{}
	srcAuth := &types.DockerAuthConfig{}

	testCopy := func(ctx context.Context, policyContext *signature.PolicyContext, destRef types.ImageReference, srcRef types.ImageReference, options *copy.Options) (copiedManifest []byte, retErr error) {
		if options.DestinationCtx.DockerAuthConfig != dstAuth {
			t.Fatal("incorrect dstAuth")
		}
		if options.SourceCtx.DockerAuthConfig != srcAuth {
			t.Fatal("incorrect srcAuth")
		}

		sentRefs = append(sentRefs, [2]types.ImageReference{destRef, srcRef})
		return []byte{}, nil
	}

	refs := []reference.NamedTagged{}

	for _, s := range []string{
		"quay.io/app-sre/managed-upgrade-operator:abc",
		"quay.io/app-sre/managed-upgrade-operator:def",
	} {
		r, err := reference.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, r.(reference.NamedTagged))
	}

	m := &repositoryMirrorManager{
		log:       log,
		copyImage: testCopy,
		dstRepo:   "test.acr",
		dstAuth:   dstAuth,
	}

	err := m.MirrorTags(context.Background(), refs, srcAuth)
	if err != nil {
		t.Fatal(err)
	}

	expected := [][2]string{
		{"test.acr/app-sre/managed-upgrade-operator:abc", "quay.io/app-sre/managed-upgrade-operator:abc"},
		{"test.acr/app-sre/managed-upgrade-operator:def", "quay.io/app-sre/managed-upgrade-operator:def"},
	}

	sentRefsComp := make([][2]string, 0, len(sentRefs))
	for _, r := range sentRefs {
		sentRefsComp = append(sentRefsComp, [2]string{r[0].DockerReference().String(), r[1].DockerReference().String()})
	}
	for _, err := range deep.Equal(sentRefsComp, expected) {
		t.Error(err)
	}

	err = testlog.AssertLoggingOutput(hook, []map[string]gomegatypes.GomegaMatcher{
		{
			"msg": gomega.Equal("mirroring quay.io/app-sre/managed-upgrade-operator:abc -> test.acr/app-sre/managed-upgrade-operator:abc"),
		},
		{
			"msg": gomega.Equal("mirroring quay.io/app-sre/managed-upgrade-operator:def -> test.acr/app-sre/managed-upgrade-operator:def"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilter(t *testing.T) {
	hook, log := testlog.New()
	srcAuth := &types.DockerAuthConfig{}

	testInspect := func(ctx context.Context, ir types.ImageReference, sc *types.SystemContext) (*types.ImageInspectInfo, error) {
		if ir.DockerReference().(reference.NamedTagged).Tag() == "bad" {
			return nil, errors.New("err")
		}
		unixt := int64(90)
		if ir.DockerReference().(reference.NamedTagged).Tag() == "def" {
			unixt = 110
		}
		t := time.Unix(unixt, 0)

		return &types.ImageInspectInfo{
			Created: &t,
		}, nil
	}

	refs := []reference.NamedTagged{}

	for _, s := range []string{
		"quay.io/app-sre/managed-upgrade-operator:abc",
		"quay.io/app-sre/managed-upgrade-operator:def",
		"quay.io/app-sre/managed-upgrade-operator:bad",
	} {
		r, err := reference.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		refs = append(refs, r.(reference.NamedTagged))
	}

	m := &repositoryMirrorManager{
		log:          log,
		inspectImage: testInspect,
	}

	gotRefs, err := m.FilterByDate(context.Background(), refs, srcAuth, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"quay.io/app-sre/managed-upgrade-operator:def",
	}
	gotRefStrings := make([]string, 0)
	for _, r := range gotRefs {
		gotRefStrings = append(gotRefStrings, r.String())
	}

	for _, err := range deep.Equal(gotRefStrings, expected) {
		t.Error(err)
	}

	err = testlog.AssertLoggingOutput(hook, []map[string]gomegatypes.GomegaMatcher{
		{
			"msg": gomega.Equal("could not inspect quay.io/app-sre/managed-upgrade-operator:bad: err"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
