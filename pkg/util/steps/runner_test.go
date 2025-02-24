package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func successfulFunc(context.Context) error { return nil }
func failingFunc(context.Context) error    { return errors.New("oh no!") }
func failingAzureError(context.Context) error {
	return errors.New("Status=403 Code=\"AuthorizationFailed\"")
}
func failingODataError(context.Context) error {
	mainError := odataerrors.NewMainError()
	mainError.SetCode(pointerutils.ToPtr("Authorization_IdentityNotFound"))
	mainError.SetMessage(pointerutils.ToPtr("The identity of the calling application could not be established."))
	e := odataerrors.NewODataError()
	e.SetErrorEscaped(mainError)
	return e
}
func alwaysFalseCondition(context.Context) (bool, error) { return false, nil }
func alwaysTrueCondition(context.Context) (bool, error)  { return true, nil }
func timingOutCondition(ctx context.Context) (bool, error) {
	time.Sleep(60 * time.Millisecond)
	return false, nil
}
func internalTimeoutCondition(ctx context.Context) (bool, error) {
	return false, context.DeadlineExceeded
}

func currentTimeFunc() time.Time {
	return time.Now()
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
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "An azure error will fail the run",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Action(failingAzureError),
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.failingAzureError]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Action pkg/util/steps.failingAzureError] encountered error: Status=403 Code=\"AuthorizationFailed\""),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "Status=403 Code=\"AuthorizationFailed\"",
		},
		{
			name: "An odata error will fail the run",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					Action(failingODataError),
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.failingODataError]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Action pkg/util/steps.failingODataError] encountered error: 400: InvalidServicePrincipalCredentials: encountered error: Authorization_IdentityNotFound: The identity of the calling application could not be established."),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "400: InvalidServicePrincipalCredentials: encountered error: Authorization_IdentityNotFound: The identity of the calling application could not be established.",
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
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.failingFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal(`step [Action pkg/util/steps.failingFunc] encountered error: oh no!`),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: `oh no!`,
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
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition pkg/util/steps.alwaysTrueCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
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
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition pkg/util/steps.alwaysFalseCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition pkg/util/steps.alwaysFalseCondition, timeout 50ms] failed but has configured 'fail=false'. Continuing. Error: context deadline exceeded"),
					"level": gomega.Equal(logrus.WarnLevel),
				},
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
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
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition pkg/util/steps.timingOutCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition pkg/util/steps.timingOutCondition, timeout 50ms] encountered error: context deadline exceeded"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "context deadline exceeded",
		},
		{
			name: "A Condition that returns a timeout error causes a different failure from a timed out Condition",
			steps: func(controller *gomock.Controller) []Step {
				return []Step{
					Action(successfulFunc),
					&conditionStep{
						f:            internalTimeoutCondition,
						fail:         true,
						pollInterval: 20 * time.Millisecond,
						timeout:      50 * time.Millisecond,
					},
					Action(successfulFunc),
				}
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition pkg/util/steps.internalTimeoutCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition pkg/util/steps.internalTimeoutCondition, timeout 50ms] encountered error: context deadline exceeded"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "context deadline exceeded",
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
					"msg":   gomega.Equal("running step [Action pkg/util/steps.successfulFunc]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("running step [Condition pkg/util/steps.alwaysFalseCondition, timeout 50ms]"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("step [Condition pkg/util/steps.alwaysFalseCondition, timeout 50ms] encountered error: context deadline exceeded"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			wantErr: "context deadline exceeded",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			h, log := testlog.New()
			steps := tt.steps(controller)

			_, err := Run(ctx, log, 25*time.Millisecond, steps, currentTimeFunc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			err = testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestStepMetricsNameFormatting(t *testing.T) {
	for _, tt := range []struct {
		desc string
		step Step
		want string
	}{
		{
			desc: "test action step naming",
			step: Action(successfulFunc),
			want: "action.successfulFunc",
		},
		{
			desc: "test condition step naming",
			step: Condition(alwaysTrueCondition, 1*time.Millisecond, true),
			want: "condition.alwaysTrueCondition",
		},
		{
			desc: "test anonymous action step naming",
			step: Action(func(context.Context) error { return nil }),
			want: "action.func1",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.step.metricsName(); got != tt.want {
				t.Errorf("incorrect step metrics name, want: %s, got: %s", tt.want, got)
			}
		})
	}
}
