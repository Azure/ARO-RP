package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

var done int32

func testExtensionsConcurrency(i int, h *codec.JsonHandle) error {
	wantDoc := &api.OpenShiftClusterDocument{
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				AROServiceKubeconfig: api.SecureBytes(fmt.Sprintf("%d", i)),
			},
		},
	}

	for {
		var b []byte
		e := codec.NewEncoderBytes(&b, h)
		err := e.Encode(wantDoc)
		if err != nil {
			return err
		}

		var doc *api.OpenShiftClusterDocument
		d := codec.NewDecoderBytes(b, h)
		err = d.Decode(&doc)
		if err != nil {
			return err
		}

		if !bytes.Equal(api.SecureBytes(fmt.Sprintf("%d", i)), doc.OpenShiftCluster.Properties.AROServiceKubeconfig) {
			return fmt.Errorf("%d: want: %s, got: %s", i,
				string(wantDoc.OpenShiftCluster.Properties.AROServiceKubeconfig),
				string(doc.OpenShiftCluster.Properties.AROServiceKubeconfig),
			)
		}

		// reading from a closed channel perturbs the scheduling too much and
		// masks unsafe errors we're currently seeing in ugorji/go master
		if atomic.LoadInt32(&done) != 0 {
			break
		}
	}

	return nil
}

func TestExtensionsConcurrency(t *testing.T) {
	const n = 100

	aead, err := encryption.NewXChaCha20Poly1305(context.Background(), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}

	h, err := NewJSONHandle(aead)
	if err != nil {
		t.Fatal(err)
	}

	errch := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			errch <- testExtensionsConcurrency(i, h)
		}(i)
	}

	time.AfterFunc(5*time.Second, func() {
		atomic.StoreInt32(&done, 1)
	})

	for i := 0; i < n; i++ {
		err = <-errch
		if err != nil {
			t.Fatal(err)
		}
	}
}
