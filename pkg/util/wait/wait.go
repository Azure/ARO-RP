package wait

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// PollImmediateWithContext will poll until a stop condition is met
func PollImmediateWithContext(interval, timeout time.Duration, condition wait.ConditionFunc, stopCh <-chan struct{}) error {
	done, err := condition()
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	return wait.PollUntil(interval, condition, stopCh)
}
