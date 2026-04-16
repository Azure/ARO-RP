package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

// digestFromReference extracts the manifest digest from a digest-based image
// reference (e.g. "registry/repo@sha256:abc123..."). Returns an error if the
// reference does not contain a digest.
func digestFromReference(reference string) (string, error) {
	if idx := strings.Index(reference, "@sha256:"); idx != -1 {
		return reference[idx+1:], nil
	}
	return "", fmt.Errorf("reference %q does not contain a digest", reference)
}

// repoFromReference extracts the repository from an image reference,
// stripping any tag or digest suffix.
func repoFromReference(reference string) string {
	if idx := strings.Index(reference, "@"); idx != -1 {
		return reference[:idx]
	}
	if idx := strings.LastIndex(reference, ":"); idx != -1 {
		if slashIdx := strings.LastIndex(reference, "/"); slashIdx < idx {
			return reference[:idx]
		}
	}
	return reference
}

// sigReference constructs the cosign .sig OCI artifact reference for a given
// repository and manifest digest (e.g. "sha256:abc123...").
func sigReference(repo, digest string) string {
	return repo + ":" + strings.Replace(digest, ":", "-", 1) + ".sig"
}

// copySigArtifact attempts to copy the cosign .sig OCI artifact for the given
// image. If the source registry has no .sig artifact for this image, the copy
// will fail — callers should treat this as non-fatal for older releases that
// were never signed.
func copySigArtifact(ctx context.Context, log *logrus.Entry, srcref, dstref string, srcauth, dstauth *types.DockerAuthConfig) error {
	dgst, err := digestFromReference(srcref)
	if err != nil {
		return err
	}

	srcRepo := repoFromReference(srcref)
	dstRepo := repoFromReference(dstref)

	sigSrc := sigReference(srcRepo, dgst)
	sigDst := sigReference(dstRepo, dgst)

	log.Debugf("mirroring sig %s -> %s", sigSrc, sigDst)
	return Copy(ctx, sigDst, sigSrc, dstauth, srcauth)
}

func Mirror(ctx context.Context, log *logrus.Entry, dstrepo, srcrelease string, dstauth, srcauth *types.DockerAuthConfig) (int, error) {
	log.Debugf("reading imagestream")
	startTime := time.Now()
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
	results := make(chan error)

	wg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			l := log.WithField("worker", i)
			for w := range ch {
				l.Debugf("mirroring %s", w.tag)
				var err error
				for retry := 0; retry < 6; retry++ {
					workTime := time.Now()
					err = Copy(ctx, w.dstreference, w.srcreference, w.dstauth, w.srcauth)
					l.WithField("duration", time.Since(workTime)).WithField("tag", w.tag).WithError(err).Debug("completed")
					if err == nil {
						break
					}
					time.Sleep(10 * time.Second)
				}
				if err != nil {
					l.WithField("tag", w.tag).WithError(err).Error("failed to mirror image after 6 retries")
				}

				if err == nil {
					sigErr := copySigArtifact(ctx, l, w.srcreference, w.dstreference, w.srcauth, w.dstauth)
					if sigErr != nil {
						l.WithField("tag", w.tag+".sig").WithError(sigErr).Debug("sig artifact not mirrored (may not exist)")
					}
				}

				results <- err
			}
			wg.Done()
		}()
	}

	go func() {
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
		close(results)
	}()

	var successful int
	var errorOccurred bool
	for err = range results {
		if err != nil {
			errorOccurred = true
		} else {
			successful++
		}
	}
	log.WithFields(logrus.Fields{
		"duration":   time.Since(startTime),
		"successful": successful,
		"total":      len(is.Spec.Tags) + 1,
	}).Infof("mirroring completed")

	if errorOccurred {
		log.Errorf("some images failed to mirror")
	}

	return successful, err
}
