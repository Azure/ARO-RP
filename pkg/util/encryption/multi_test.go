package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_encryption "github.com/Azure/ARO-RP/pkg/util/mocks/encryption"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestOpen(t *testing.T) {
	mockInput := []byte("fakeInput")

	type test struct {
		name       string
		mocks      func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD)
		wantResult []byte
		wantErr    string
	}

	for _, tt := range []*test{
		{
			name: "first opener succeeds, do not try second",
			mocks: func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD) {
				firstOpener.EXPECT().Open(mockInput).Return([]byte("result from the first opener"), nil)
			},
			wantResult: []byte("result from the first opener"),
		},
		{
			name: "first opener errors, but second succeeds",
			mocks: func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD) {
				firstOpener.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the first opener"))
				secondOpener.EXPECT().Open(mockInput).Return([]byte("result from the second opener"), nil)
			},
			wantResult: []byte("result from the second opener"),
		},
		{
			name: "all openers error",
			mocks: func(firstOpener *mock_encryption.MockAEAD, secondOpener *mock_encryption.MockAEAD) {
				firstOpener.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the first opener"))
				secondOpener.EXPECT().Open(mockInput).Return(nil, errors.New("fake error from the second opener"))
			},
			wantErr: "no openers succeeded:\n\t* first: fake error from the first opener\n\t* second: fake error from the second opener",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)

			firstOpener := mock_encryption.NewMockAEAD(controller)
			firstOpener.EXPECT().Name().Return("first").AnyTimes()
			secondOpener := mock_encryption.NewMockAEAD(controller)
			secondOpener.EXPECT().Name().Return("second").AnyTimes()

			multi := multi{
				openers: []AEAD{
					firstOpener,
					secondOpener,
				},
			}

			tt.mocks(firstOpener, secondOpener)

			b, err := multi.Open(mockInput)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if b != nil && !reflect.DeepEqual(tt.wantResult, b) ||
				b == nil && tt.wantResult != nil {
				t.Error(b)
			}
		})
	}
}

func TestMultiName(t *testing.T) {
	controller := gomock.NewController(t)

	firstOpener := mock_encryption.NewMockAEAD(controller)
	firstOpener.EXPECT().Name().Return("first").AnyTimes()
	secondOpener := mock_encryption.NewMockAEAD(controller)
	secondOpener.EXPECT().Name().Return("second").AnyTimes()

	multi := multi{
		sealer: firstOpener,
		openers: []AEAD{
			firstOpener,
			secondOpener,
		},
	}

	expected := "Multi(sealer=first, openers=first,second)"

	assert.Equal(t, expected, multi.Name())
}
