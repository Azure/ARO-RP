package steps

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
	eg := &sync.WaitGroup{}
	errlock := &sync.Mutex{}
	errs := []error{}
	for _, curStep := range s.s {
		eg.Go(func() {
			err := curStep.run(ctx, log)
			errlock.Lock()
			defer errlock.Unlock()
			errs = append(errs, err)
		})
	}

	eg.Wait()
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
