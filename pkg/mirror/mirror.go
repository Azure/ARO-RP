package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

func Copy(ctx context.Context, dstreference, srcreference string, dstauth, srcauth *types.DockerAuthConfig) error {
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
		// Images that we mirror shouldn't change, so we can use the
		// optimisation that checks if the source and destination manifests are
		// equal before attempting to push it (and sending no blobs because
		// they're all already there)
		OptimizeDestinationImageAlreadyExists: true,
	})

	return err
}

// This will return repo and image name, preserving path
// Ex:
// repo      destrepo.io
// reference azurecr.io/some/path/to/image:tag
// returns:  destrepo.io/some/path/to/image:tag
func Dest(repo, reference string) string {
	return repo + reference[strings.IndexByte(reference, '/'):]
}

// This will return repo / image name.
// Ex:
// repo      destrepo.io
// reference azurecr.io/some/path/to/image:tag
// returns:  destrepo.io/image:tag
func DestLastIndex(repo, reference string) string {
	return repo + reference[strings.LastIndex(reference, "/"):]
}

func Mirror(ctx context.Context, log *logrus.Entry, dstrepo, srcrelease string, dstauth, srcauth *types.DockerAuthConfig) (int, error) {
	log.Debugf("reading imagestream")
	is, err := getReleaseImageStream(ctx, srcrelease, srcauth)
	if err != nil {
		log.WithError(err).Errorf("failed to read imagestream")
		return 0, err
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
	var successfulImages atomic.Uint32
	errorOccurred.Store(false)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for w := range ch {
				log.Debugf("mirroring %s", w.tag)
				var err error
				for retry := 0; retry < 6; retry++ {
					err = Copy(ctx, w.dstreference, w.srcreference, w.dstauth, w.srcauth)
					if err == nil {
						break
					}
					time.Sleep(10 * time.Second)
				}
				if err != nil {
					log.WithField("tag", w.tag).WithError(err).Error("failed to mirror image after 6 retries")
					errorOccurred.Store(true)
				}
				successfulImages.Add(1)
			}
			wg.Done()
		}()
	}

	log.Printf("mirroring %d image(s)", len(is.Spec.Tags)+1)

	ch <- &work{
		tag:          "release",
		dstreference: Dest(dstrepo, srcrelease),
		srcreference: srcrelease,
		dstauth:      dstauth,
		srcauth:      srcauth,
	}

	for _, tag := range is.Spec.Tags {
		ch <- &work{
			tag:          tag.Name,
			dstreference: Dest(dstrepo, tag.From.Name),
			srcreference: tag.From.Name,
			dstauth:      dstauth,
			srcauth:      srcauth,
		}
	}

	close(ch)
	wg.Wait()

	if errorOccurred.Load().(bool) {
		return int(successfulImages.Load()), fmt.Errorf("an error occurred")
	}

	return int(successfulImages.Load()), nil
}
