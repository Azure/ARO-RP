package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"log"

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var ErrMirror = errors.New("an error occurred mirroring an image")

type Manager struct {
	env env.Core
	log *logrus.Entry

	dstAcr  string
	dstAuth *types.DockerAuthConfig
}

func New(env env.Core, log *logrus.Entry, dstAcr string, dstAuth *types.DockerAuthConfig) *Manager {
	return &Manager{
		env: env,
		log: log,

		dstAcr:  dstAcr,
		dstAuth: dstAuth,
	}
}

func (m *Manager) MirrorOpenShiftVersion(ctx context.Context, srcAuth *types.DockerAuthConfig, minorVersionToMirror *version.Version, doNotMirrorTags map[string]struct{}) error {
	m.log.Printf("reading release graph for OpenShift %s", minorVersionToMirror.MinorVersion())

	releases, err := AddFromGraph(minorVersionToMirror)
	if err != nil {
		return err
	}

	return m.doOpenShiftMirror(ctx, srcAuth, releases, doNotMirrorTags)
}

func (m *Manager) doOpenShiftMirror(ctx context.Context, srcAuth *types.DockerAuthConfig, releases []Node, doNotMirrorTags map[string]struct{}) error {
	errorOccurred := false

	for _, release := range releases {
		if _, ok := doNotMirrorTags[release.Version]; ok {
			log.Printf("skipping mirror of release %s", release.Version)
			continue
		}

		log.Printf("mirroring OpenShift release %s", release.Version)
		err := Mirror(ctx, m.log, m.dstAcr, release.Payload, m.dstAuth, srcAuth)
		if err != nil {
			errorOccurred = true
		}
	}

	if errorOccurred {
		return ErrMirror
	}
	return nil
}

func (m *Manager) MirrorImageRefs(ctx context.Context, srcAuth *types.DockerAuthConfig, images []string) error {
	errorOccurred := false

	for _, ref := range images {
		dst := Dest(m.dstAcr, ref)
		m.log.Printf("mirroring %s -> %s", ref, dst)

		err := Copy(ctx, dst, ref, m.dstAuth, srcAuth)
		if err != nil {
			errorOccurred = true
		}
	}

	if errorOccurred {
		return ErrMirror
	}
	return nil
}
