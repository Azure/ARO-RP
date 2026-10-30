package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/storage"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestLoadPersisted(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	storage := mock_storage.NewMockManager(ctrl)
	storage.EXPECT().BlobService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("general error"))

	_, log := testlog.LogForTesting(t)
	m := &manager{
		log:     log,
		storage: storage,
	}

	_, err := m.LoadPersisted(ctx, "test-rg", "TEST-ACCOUNT")
	utilerror.AssertErrorMessage(t, err, "general error")
}
