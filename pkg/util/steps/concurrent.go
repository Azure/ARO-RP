package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

func Concurrent(name string, s []Step) Step {
	return concurrentStep{
		name: name,
		s:    s,
	}
}

type concurrentStep struct {
	name string
	s    []Step
}

func (s concurrentStep) run(ctx context.Context, log *logrus.Entry) error {
	wg := &sync.WaitGroup{}
	errChan := make(chan error, len(s.s))
	for _, curStep := range s.s {
		wg.Go(func() {
			defer func() {
				if err := recover(); err != nil {
					errChan <- fmt.Errorf("panic in step '%s' : %v", curStep.String(), err)
				}
			}()

			err := curStep.run(ctx, log)
			if err != nil {
				errChan <- err
			}
		})
	}

	wg.Wait()
	close(errChan)

	errs := []error{}
	for err := range errChan {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (s concurrentStep) String() string {
	stepNames := []string{}

	for _, curStep := range s.s {
		na := strings.Trim(curStep.String(), "[]")
		sho := shortName(na)
		stepNames = append(stepNames, sho)
	}

	return fmt.Sprintf("[Action %s [%s]]", s.name, strings.Join(stepNames, ", "))
}

func (s concurrentStep) metricsName() string {
	return fmt.Sprintf("action.%s", s.name)
}
