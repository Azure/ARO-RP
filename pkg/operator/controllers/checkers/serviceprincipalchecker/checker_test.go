package serviceprincipalchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/golang/mock/gomock"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type mockValidator struct {
	err error
}

func (v mockValidator) Validate(token *adal.ServicePrincipalToken) error {
	return v.err
}

type mockTokenGetter struct {
	err error
}

func (g mockTokenGetter) Get(ctx context.Context, azEnv *azureclient.AROEnvironment) (token *adal.ServicePrincipalToken, err error) {
	return nil, g.err
}
func TestCheck2(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name          string
		validatorErr  error
		wantErr       string
		credGetterErr error
	}{
		{
			name: "valid service principal",
		},
		{
			name:          "could not get service principal credentials",
			credGetterErr: errors.New("fake credentials get error"),
			wantErr:       "fake credentials get error",
		},
		{
			name:         "could not validate",
			validatorErr: errors.New("some error"),
			wantErr:      "some error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			credentialsGetter := mockTokenGetter{err: tt.credGetterErr}
			sp := &checker{
				log:        log,
				credGetter: credentialsGetter,
			}

			sp.spValidator = mockValidator{err: tt.validatorErr}

			err := sp.Check(ctx, azuretypes.PublicCloud.Name())
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%s\n !=\n%s", err, tt.wantErr)
			}
		})
	}
}
