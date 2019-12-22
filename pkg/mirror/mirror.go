package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	"github.com/sirupsen/logrus"
)

func copyImage(ctx context.Context, dstreference, srcreference string, dstauth, srcauth *types.DockerAuthConfig) error {
	policyctx, err := signature.NewPolicyContext(&signature.Policy{
		Default: signature.PolicyRequirements{
			signature.NewPRInsecureAcceptAnything(),
		},
	})
	if err != nil {
		return err
	}

	src, err := docker.ParseReference("//" + srcreference)
	if err != nil {
		return err
	}

	dst, err := docker.ParseReference("//" + dstreference)
	if err != nil {
		return err
	}

	_, err = copy.Image(ctx, policyctx, dst, src, &copy.Options{
		SourceCtx: &types.SystemContext{
			DockerAuthConfig: srcauth,
		},
		DestinationCtx: &types.SystemContext{
			DockerAuthConfig: dstauth,
		},
	})

	return err
}

func dst(repo, reference string) string {
	return repo + reference[strings.IndexByte(reference, '/'):]
}

func Mirror(ctx context.Context, log *logrus.Entry, dstrepo, srcrelease string, dstauth, srcauth *types.DockerAuthConfig) error {
	log.Printf("reading imagestream from %s", srcrelease)
	is, err := getReleaseImageStream(ctx, srcrelease, srcauth)
	if err != nil {
		return err
	}

	type work struct {
		tag          string
		dstreference string
		srcreference string
		dstauth      *types.DockerAuthConfig
		srcauth      *types.DockerAuthConfig
	}

	ch := make(chan *work)
	wg := &sync.WaitGroup{}
	var errorOccurred atomic.Value
	errorOccurred.Store(false)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for w := range ch {
				log.Printf("mirroring %s", w.tag)
				err := copyImage(ctx, w.dstreference, w.srcreference, w.dstauth, w.srcauth)
				if err != nil {
					log.Errorf("%s: %s\n", w.tag, err)
					errorOccurred.Store(true)
				}
			}
			wg.Done()
		}()
	}

	log.Printf("mirroring %d image(s)", len(is.Spec.Tags)+1)

	ch <- &work{
		tag:          "release",
		dstreference: dst(dstrepo, srcrelease),
		srcreference: srcrelease,
		dstauth:      dstauth,
		srcauth:      srcauth,
	}

	for _, tag := range is.Spec.Tags {
		ch <- &work{
			tag:          tag.Name,
			dstreference: dst(dstrepo, tag.From.Name),
			srcreference: tag.From.Name,
			dstauth:      dstauth,
			srcauth:      srcauth,
		}
	}

	close(ch)
	wg.Wait()

	if errorOccurred.Load().(bool) {
		return fmt.Errorf("an error occurred")
	}

	return nil
}
