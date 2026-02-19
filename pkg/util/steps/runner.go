package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
)

// FriendlyName returns a "friendly" stringified name of the given func. This
// consists of removing the ARO base package name (so it produces pkg/foobar
// instead of github.com/Azure/ARO-RP/pkg/foobar) and removing the -fm suffix
// from Golang struct methods.
func FriendlyName(f interface{}) string {
	return strings.TrimPrefix(strings.TrimSuffix(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), "-fm"), "github.com/Azure/ARO-RP/")
}

// ShortName returns a shorter "friendly" stringified name of the given
// function. This should be used when all calls are either uniquely named or
// methods on the same struct to avoid the need for disambiguation.
func ShortName(f any) string {
	return shortName(FriendlyName(f))
}

func shortName(fullName string) string {
	sepCheck := func(c rune) bool {
		return c == '/' || c == '.'
	}

	fields := strings.FieldsFunc(strings.TrimSpace(fullName), sepCheck)

	if size := len(fields); size > 0 {
		return fields[size-1]
	}
	return fullName
}

// Step is the interface for steps that Runner can execute.
type Step interface {
	run(ctx context.Context, log *logrus.Entry) error
	String() string
	metricsName() string
}

// Run executes the provided steps in order until one fails or all steps
// are completed. Errors from failed steps are returned directly.
// time cost for each step run will be recorded for metrics usage
func Run(ctx context.Context, log *logrus.Entry, pollInterval time.Duration, steps []Step, now func() time.Time, managedRGName string) (map[string]int64, error) {
	stepTimeRun := make(map[string]int64)
	for _, step := range steps {
		log.Infof("running step %s", step)

		startTime := time.Now()
		err := step.run(ctx, log)
		if err != nil {
			if azureerrors.IsUnauthorizedClientError(err) ||
				azureerrors.IsInvalidSecretError(err) {
				err = api.NewCloudError(
					http.StatusBadRequest,
					api.CloudErrorCodeInvalidServicePrincipalCredentials,
					"encountered error",
					err.Error())
			} else if azureerrors.HasAuthorizationFailedError(err) {
				errCode := api.CloudErrorCodeInvalidServicePrincipalCredentials
				if managedRGName != "" && azureerrors.IsManagedResourceGroupError(err, managedRGName) {
					errCode = api.CloudErrorCodeInvalidResourceProviderPermissions
				}
				err = api.NewCloudError(
					http.StatusBadRequest,
					errCode,
					"encountered error",
					err.Error())
			} else if oDataError := (&msgraph_errors.ODataError{}); errors.As(err, &oDataError) {
				if *oDataError.GetErrorEscaped().GetCode() == "Authorization_IdentityNotFound" {
					err = api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodeInvalidServicePrincipalCredentials,
						"encountered error",
						fmt.Sprintf(
							"%s: %s",
							*oDataError.GetErrorEscaped().GetCode(),
							*oDataError.GetErrorEscaped().GetMessage()))
				} else {
					spew.Fdump(log.Writer(), oDataError.GetErrorEscaped())
				}
			}
			log.Errorf("step %s encountered error: %s", step, err.Error())
			return nil, err
		}

		if now != nil {
			currentTime := now()
			stepTimeRun[step.metricsName()] = int64(currentTime.Sub(startTime).Seconds())
		}
	}
	return stepTimeRun, nil
}
