package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// friendlyName returns a "friendly" stringified name of the given func.
func friendlyName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// Step is the interface for steps that Runner can execute.
type Step interface {
	run(ctx context.Context, log *logrus.Entry) error
	setPollInterval(time.Duration)
	setTimeout(time.Duration)
	String() string
}

// Run executes the provided steps in order until one fails or all steps
// are completed. Errors from failed steps are returned directly.
func Run(ctx context.Context, log *logrus.Entry, pollInterval time.Duration, globalTimeout time.Duration, steps []Step, testHook StageHook) error {
	for _, step := range steps {
		log.Infof("running step %s", step)

		// Pre-Step hook
		err := testHook.PreRun(ctx, step)
		if err != nil {
			log.Errorf("step %s pre-run encountered error: %s", step, err.Error())
			return err
		}

		step.setPollInterval(pollInterval)

		// If we have set a global timeout, apply it to the step. This is mostly
		// useful for testing.
		if globalTimeout != 0 {
			step.setTimeout(globalTimeout)
		}

		err = step.run(ctx, log)
		if err != nil {
			errmsg := fmt.Errorf("step %s encountered error: %w", step, err)
			log.Error(errmsg)
			return errmsg
		}
		// Post-Step hook
		err = testHook.PostRun(ctx, step)
		if err != nil {
			log.Errorf("step %s post-run encountered error: %s", step, err.Error())
			return err
		}
	}
	return nil
}
