package testpoller

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "time"

// Poll calls the poller function every pollInterval until maximumWait or poller
// returns true. If poller returns an error, it's returned directly.
func Poll(maximumWait time.Duration, pollInterval time.Duration, poller func() (bool, error)) error {
	for start := time.Now(); time.Since(start) < maximumWait; {
		res, err := poller()
		if err != nil {
			return err
		}
		if res {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return nil
}
