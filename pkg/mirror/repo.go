package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

type repositoryMirrorManager struct {
	log *logrus.Entry

	getRepositoryTags func(context.Context, *types.SystemContext, types.ImageReference) ([]string, error)
	copyImage         func(context.Context, *signature.PolicyContext, types.ImageReference, types.ImageReference, *copy.Options) ([]byte, error)
	inspectImage      func(context.Context, types.ImageReference, *types.SystemContext) (*types.ImageInspectInfo, error)
	dstAuth           *types.DockerAuthConfig
	dstRepo           string
}

type RepositoryMirrorManager interface {
	List(context.Context, string, *types.DockerAuthConfig) ([]reference.NamedTagged, error)
	FilterByDate(context.Context, []reference.NamedTagged, *types.DockerAuthConfig, time.Time) ([]reference.NamedTagged, error)
	MirrorTags(context.Context, []reference.NamedTagged, *types.DockerAuthConfig) error
}

func NewRepositoryMirrorManager(dstRepo string, dstAuth *types.DockerAuthConfig) RepositoryMirrorManager {
	return &repositoryMirrorManager{
		getRepositoryTags: docker.GetRepositoryTags,
		copyImage:         copy.Image,
		dstRepo:           dstRepo,
		dstAuth:           dstAuth,
		inspectImage: func(ctx context.Context, ir types.ImageReference, sc *types.SystemContext) (*types.ImageInspectInfo, error) {
			i, err := ir.NewImage(ctx, sc)
			if err != nil {
				return nil, err
			}
			defer i.Close()
			return i.Inspect(ctx)
		},
	}
}

func (m *repositoryMirrorManager) List(ctx context.Context, srcReference string, srcauth *types.DockerAuthConfig) ([]reference.NamedTagged, error) {
	ref, err := docker.ParseReference("//" + srcReference)
	if err != nil {
		return nil, err
	}

	tags, err := m.getRepositoryTags(ctx, &types.SystemContext{
		DockerAuthConfig: srcauth,
	}, ref)
	if err != nil {
		return nil, err
	}

	r := make([]reference.NamedTagged, 0, len(tags))

	for _, tag := range tags {
		wt, err := reference.WithTag(ref.DockerReference(), tag)
		if err != nil {
			return nil, err
		}
		r = append(r, wt)
	}

	return r, nil
}

func (m *repositoryMirrorManager) MirrorTags(ctx context.Context, refs []reference.NamedTagged, srcAuth *types.DockerAuthConfig) error {
	policyctx, err := signature.NewPolicyContext(&signature.Policy{
		Default: signature.PolicyRequirements{
			signature.NewPRInsecureAcceptAnything(),
		},
	})
	if err != nil {
		return err
	}

	for _, r := range refs {
		src, err := docker.NewReference(r)
		if err != nil {
			return err
		}

		dstPath := Dest(m.dstRepo, r.String())
		dst, err := docker.ParseReference("//" + dstPath)
		if err != nil {
			return err
		}

		m.log.Printf("mirroring %s -> %s", r, dstPath)
		_, err = m.copyImage(ctx, policyctx, dst, src, &copy.Options{
			SourceCtx: &types.SystemContext{
				DockerAuthConfig: srcAuth,
			},
			DestinationCtx: &types.SystemContext{
				DockerAuthConfig: m.dstAuth,
			},
			// Images that we mirror shouldn't change, so we can use the
			// optimisation that checks if the source and destination manifests are
			// equal before attempting to push it (and sending no blobs because
			// they're all already there)
			OptimizeDestinationImageAlreadyExists: true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *repositoryMirrorManager) FilterByDate(ctx context.Context, refs []reference.NamedTagged, srcAuth *types.DockerAuthConfig, notBefore time.Time) ([]reference.NamedTagged, error) {
	allowedRefs := make([]reference.NamedTagged, 0)

	fetchInfo := func(r reference.NamedTagged) error {
		src, err := docker.NewReference(r)
		if err != nil {
			return err
		}

		inspectInfo, err := m.inspectImage(ctx, src, &types.SystemContext{
			DockerAuthConfig: srcAuth,
		})
		if err != nil {
			m.log.Warnf("could not inspect %s: %s", r.String(), err)
			return nil
		}

		if inspectInfo.Created.UTC().After(notBefore) {
			allowedRefs = append(allowedRefs, r)
		}
		return nil
	}

	for _, r := range refs {
		err := fetchInfo(r)
		if err != nil {
			return nil, err
		}
	}

	return allowedRefs, nil
}

func MirrorTagsByFilteredDate(ctx context.Context, mgr RepositoryMirrorManager, srcReference string, srcAuth *types.DockerAuthConfig, filterSince time.Time) error {
	tags, err := mgr.List(ctx, srcReference, srcAuth)
	if err != nil {
		return err
	}

	filteredTags, err := mgr.FilterByDate(ctx, tags, srcAuth, filterSince)
	if err != nil {
		return err
	}

	return mgr.MirrorTags(ctx, filteredTags, srcAuth)
}
