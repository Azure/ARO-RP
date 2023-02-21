package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	msgraph_errors "github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	"github.com/sirupsen/logrus"
)

// FriendlyName returns a "friendly" stringified name of the given func.
func FriendlyName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
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
func Run(ctx context.Context, log *logrus.Entry, pollInterval time.Duration, steps []Step, now func() time.Time) (map[string]int64, error) {
	stepTimeRun := make(map[string]int64)
	for _, step := range steps {
		log.Infof("running step %s", step)

		startTime := time.Now()
		err := step.run(ctx, log)

		if err != nil {
			log.Errorf("step %s encountered error: %s", step, err.Error())
			if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
				spew.Fdump(log.Writer(), oDataError.GetError())
			}
			return nil, err
		}

		if now != nil {
			currentTime := now()
			stepTimeRun[step.metricsName()] = int64(currentTime.Sub(startTime).Seconds())
		}
	}
	return stepTimeRun, nil
}
