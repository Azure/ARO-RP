package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Rebuilder interface {
	Rebuild() error
}

type multiRebuilder []Rebuilder

func (m multiRebuilder) Rebuild() error {
	for _, r := range m {
		if r == nil {
			continue
		}

		err := r.Rebuild()
		if err != nil {
			return err
		}
	}

	return nil
}

func NewMultiRebuilder(rebuilders ...Rebuilder) Rebuilder {
	return multiRebuilder(rebuilders)
}
