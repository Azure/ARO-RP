package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/pkg/blobinfocache/memory"
	"github.com/containers/image/v5/types"

	imagev1 "github.com/openshift/api/image/v1"
)

func getReleaseImageStream(ctx context.Context, reference string, auth *types.DockerAuthConfig) (*imagev1.ImageStream, error) {
	systemctx := &types.SystemContext{
		DockerAuthConfig: auth,
	}

	ref, err := docker.ParseReference("//" + reference)
	if err != nil {
		return nil, err
	}

	img, err := ref.NewImage(ctx, systemctx)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	src, err := ref.NewImageSource(ctx, systemctx)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	layerInfos := img.LayerInfos()
	for i := len(layerInfos) - 1; i >= 0; i-- {
		rc, _, err := src.GetBlob(ctx, layerInfos[i], memory.New())
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		br := bufio.NewReader(rc)

		b, err := br.Peek(2)
		if err != nil {
			return nil, err
		}

		var r io.Reader = br
		if bytes.Equal(b, []byte("\x1f\x8b")) {
			gr, err := gzip.NewReader(br)
			if err != nil {
				return nil, err
			}
			defer gr.Close()

			r = gr
		}

		tr := tar.NewReader(r)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			switch h.Name {
			case "release-manifests/image-references":
				var is *imagev1.ImageStream

				err = json.NewDecoder(tr).Decode(&is)
				if err != nil {
					return nil, err
				}

				return is, nil
			}
		}
	}

	return nil, fmt.Errorf("image references not found")
}
