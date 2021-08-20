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
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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
		wantEntries []map[string]types.GomegaMatcher
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
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
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
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal(`step [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc] encountered error: oh no!`),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: `oh no!`,
		},
		{
			name: "An AuthorizationRefreshingAction that fails but is retried successfully will allow a successful run",
			steps: func(controller *gomock.Controller) []Step {
				refreshable := mock_refreshable.NewMockAuthorizer(controller)
				refreshable.EXPECT().
					RefreshWithContext(gomock.Any(), gomock.Any()).
					Return(true, nil)

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
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.MatchRegexp(`running step \[AuthorizationRefreshingAction \[Action github.com/Azure/ARO-RP/pkg/util/steps\.TestStepRunner\..*.\.1]]`),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal(`TEST#GET: oops: StatusCode=403 -- Original Error: Code="AuthorizationFailed" Message="failed"`),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "A successful condition will allow steps to continue",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Condition(alwaysTrueCondition, 50*time.Millisecond, true),
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysTrueCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "A failed condition with fail=false will allow steps to continue",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Condition(alwaysFalseCondition, 50*time.Millisecond, false),
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysFalseCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysFalseCondition, timeout 50ms] failed but has configured 'fail=false'. Continuing. Error: timed out waiting for the condition"),
					"level": gomega.Equal(logrus.WarnLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
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
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failsWithAzureError]]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal(`step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failsWithAzureError]] encountered error: TEST#GET: oops: StatusCode=403 -- Original Error: Code="AuthorizationFailed" Message="failed"`),
					"level": gomega.Equal(logrus.ErrorLevel),
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
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc]]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal(`step [AuthorizationRefreshingAction [Action github.com/Azure/ARO-RP/pkg/util/steps.failingFunc]] encountered error: oh no!`),
					"level": gomega.Equal(logrus.ErrorLevel),
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
						fail:         true,
						pollInterval: 20 * time.Millisecond,
						timeout:      50 * time.Millisecond,
					},
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.timingOutCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition github.com/Azure/ARO-RP/pkg/util/steps.timingOutCondition, timeout 50ms] encountered error: timed out waiting for the condition"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "timed out waiting for the condition",
		},
		{
			name: "A Condition that does not return true in the timeout time causes a failure",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Condition(alwaysFalseCondition, 50*time.Millisecond, true),
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action github.com/Azure/ARO-RP/pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysFalseCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition github.com/Azure/ARO-RP/pkg/util/steps.alwaysFalseCondition, timeout 50ms] encountered error: timed out waiting for the condition"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "timed out waiting for the condition",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			h, log := testlog.New()
			steps := tt.steps(controller)

			err := Run(ctx, log, 25*time.Millisecond, steps)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			err = testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
