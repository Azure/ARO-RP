package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	mock_refreshable "github.com/Azure/ARO-RP/pkg/util/mocks/refreshable"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func successfulFunc(context.Context) error { return nil }
func failingFunc(context.Context) error    { return errors.New("oh no!") }
func failsWithAzureError(ctx context.Context) error {
	return autorest.DetailedError{
		Method:      "GET",
		PackageType: "TEST",
		Message:     "oops",
		StatusCode:  403,
		Original: &azure.ServiceError{
			Code:    "AuthorizationFailed",
			Message: "failed",
		},
	}
}
func alwaysFalseCondition(context.Context) (bool, error) { return false, nil }
func alwaysTrueCondition(context.Context) (bool, error)  { return true, nil }
func timingOutCondition(ctx context.Context) (bool, error) {
	time.Sleep(60 * time.Millisecond)
	return false, nil
}

func TestStepRunner(t *testing.T) {
	for _, tt := range []struct {
		name        string
		steps       func(*gomock.Controller) []Step
		wantEntries []testlog.ExpectedLogEntry
		wantErr     string
	}{
		{
			name: "All successful Actions will have a successful run",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Action(successfulFunc),
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "A failing Action will fail the run",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Action(failingFunc),
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: `step [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc] encountered error: oh no!`,
					Level:   logrus.ErrorLevel,
				},
			},
			wantErr: `oh no!`,
		},
		{
			name: "An AuthorizationRefreshingAction that fails but is retried successfully will allow a successful run",
			steps: func(controller *gomock.Controller) []Step {
				refreshable := mock_refreshable.NewMockAuthorizer(controller)
				refreshable.EXPECT().
					RefreshWithContext(gomock.Any()).
					Return(nil)

				errsRemaining := 1
				action := Action(func(context.Context) error {
					if errsRemaining > 0 {
						errsRemaining--
						return autorest.DetailedError{
							Method:      "GET",
							PackageType: "TEST",
							Message:     "oops",
							StatusCode:  403,
							Original: &azure.ServiceError{
								Code:    "AuthorizationFailed",
								Message: "failed",
							},
						}
					}
					return nil
				})

				errorsOnce := &authorizationRefreshingActionStep{
					step:         action,
					authorizer:   refreshable,
					retryTimeout: 50 * time.Millisecond,
					pollInterval: 25 * time.Millisecond,
				}

				return []Step{
					Action(successfulFunc),
					errorsOnce,
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					MessageRegex: `running step \[AuthorizationRefreshingAction \[Action github.com/Azure/ARO-RP/pkg/util/steps\.TestStepRunner\..*.\.1]]`,
					Level:        logrus.InfoLevel,
				},
				{
					Message: `TEST#GET: oops: StatusCode=403 -- Original Error: Code="AuthorizationFailed" Message="failed"`,
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "A successful condition will allow steps to continue",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Condition(alwaysTrueCondition, 50*time.Millisecond),
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysTrueCondition, timeout 50ms]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "AuthorizationRefreshingAction will not refresh once it is timed out",
			steps: func(controller *gomock.Controller) []Step {
				// We time out immediately, so we won't actually try and refresh
				refreshable := mock_refreshable.NewMockAuthorizer(controller)
				return []Step{
					Action(successfulFunc),
					&authorizationRefreshingActionStep{
						step:         Action(failsWithAzureError),
						authorizer:   refreshable,
						retryTimeout: 1 * time.Nanosecond,
					},
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failsWithAzureError]]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: `step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failsWithAzureError]] encountered error: TEST#GET: oops: StatusCode=403 -- Original Error: Code="AuthorizationFailed" Message="failed"`,
					Level:   logrus.ErrorLevel,
				},
			},
			wantErr: `TEST#GET: oops: StatusCode=403 -- Original Error: Code="AuthorizationFailed" Message="failed"`,
		},
		{
			name: "AuthorizationRefreshingAction will not refresh on a real failure",
			steps: func(controller *gomock.Controller) []Step {
				refreshable := mock_refreshable.NewMockAuthorizer(controller)
				return []Step{
					Action(successfulFunc),
					AuthorizationRefreshingAction(refreshable, Action(failingFunc)),
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc]]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: `step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc]] encountered error: oh no!`,
					Level:   logrus.ErrorLevel,
				},
			},
			wantErr: `oh no!`,
		},
		{
			name: "A timed out Condition causes a failure",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					&conditionStep{
						f:            timingOutCondition,
						pollInterval: 20 * time.Millisecond,
						timeout:      50 * time.Millisecond,
					},
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.timingOutCondition, timeout 50ms]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "step [Condition github.com/Azure/ARO-RP/pkg/util/steps.timingOutCondition, timeout 50ms] encountered error: timed out waiting for the condition",
					Level:   logrus.ErrorLevel,
				},
			},
			wantErr: "timed out waiting for the condition",
		},
		{
			name: "A Condition that does not return true in the timeout time causes a failure",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Condition(alwaysFalseCondition, 50*time.Millisecond),
					Action(successfulFunc),
				}
			},
			wantEntries: []testlog.ExpectedLogEntry{
				{
					Message: "running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysFalseCondition, timeout 50ms]",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysFalseCondition, timeout 50ms] encountered error: timed out waiting for the condition",
					Level:   logrus.ErrorLevel,
				},
			},
			wantErr: "timed out waiting for the condition",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			h, log := testlog.NewCapturingLogger()
			steps := tt.steps(controller)

			err := Run(ctx, log, 25*time.Millisecond, steps)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			for _, e := range testlog.AssertLoggingOutput(h, tt.wantEntries) {
				t.Error(e)
			}
		})
	}
}
