package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/storage"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestLoadPersisted(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_storage.MockManager, *mock_env.MockInterface, *mock_keyvault.MockManager)
		wantErr string
	}{
		{
			name: "loadPersisted returns an error other than the chacha20poly1305 one",
			mocks: func(storage *mock_storage.MockManager, env *mock_env.MockInterface, kv *mock_keyvault.MockManager) {
				storage.EXPECT().BlobService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("general error"))
			},
			wantErr: "general error",
		},
		{
			name: "loadPersisted returns a chacha20poly1305 error",
			mocks: func(storage *mock_storage.MockManager, env *mock_env.MockInterface, kv *mock_keyvault.MockManager) {
				storage.EXPECT().BlobService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("chacha20poly1305: message authentication failed"))
				env.EXPECT().ServiceKeyvault().Return(kv)
				kv.EXPECT().GetBase64Secret(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			rg := "test-rg"
			account := "TEST-ACCOUNT"
			env := mock_env.NewMockInterface(ctrl)
			storage := mock_storage.NewMockManager(ctrl)
			kv := mock_keyvault.NewMockManager(ctrl)

			tt.mocks(storage, env, kv)

			m := &manager{
				log:     logrus.NewEntry(logrus.StandardLogger()),
				storage: storage,
				env:     env,
			}

			_, err := m.LoadPersisted(ctx, rg, account)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
