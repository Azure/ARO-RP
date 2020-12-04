package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type dynValidateTest struct {
	name      string
	validator func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument, refreshable.Authorizer) validate.OpenShiftClusterDynamicValidator
	wantErr   string
}

type fakeValidator struct {
	err error
}

func (fv *fakeValidator) Dynamic(context.Context) error {
	return fv.err
}

func TestDynamicValidate(t *testing.T) {
	for _, tt := range []dynValidateTest{
		{
			name: "dynamic validator success does not trigger an error",
			validator: func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument, refreshable.Authorizer) validate.OpenShiftClusterDynamicValidator {
				return &fakeValidator{
					err: nil,
				}
			},
		},
		{
			name:    "dynamic validator failure returns the error",
			wantErr: "oh no",
			validator: func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument, refreshable.Authorizer) validate.OpenShiftClusterDynamicValidator {
				return &fakeValidator{
					err: errors.New("oh no"),
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, log := testlog.New()

			controller := gomock.NewController(t)
			defer controller.Finish()
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().DeploymentMode().AnyTimes().Return(deployment.Development)
			_env.EXPECT().Zones(gomock.Any()).AnyTimes().Return([]string{"useast2a"}, nil)
			_env.EXPECT().Domain().AnyTimes().Return("aroapp.io")

			doc := &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			}

			c := &manager{
				log:                 log,
				doc:                 doc,
				env:                 _env,
				newDynamicValidator: tt.validator,
			}

			err := c.validateResources(ctx)

			var errCheck string
			if err != nil {
				errCheck = err.Error()
			}
			errs := deep.Equal(tt.wantErr, errCheck)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
