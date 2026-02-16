package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// AllIteratorsConsumed returns whether all fake changefeeds have consumed their
// full contents
func (c *FakeSubscriptionDocumentClient) AllIteratorsConsumed() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, i := range c.changeFeedIterators {
		if !i.done {
			return false
		}
	}
	return true
}
