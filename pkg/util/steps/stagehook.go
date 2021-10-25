package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
)

type StageHook interface {
	PreRun(context.Context, Step) error
	PostRun(context.Context, Step) error
}

// no-op StageHook for production
type nilStageHook struct{}

func (d *nilStageHook) PreRun(ctx context.Context, s Step) error {
	return nil
}

func (d *nilStageHook) PostRun(ctx context.Context, s Step) error {
	return nil
}

func NewNilStageHook() StageHook {
	return &nilStageHook{}
}

// test hook that runs pre and post hooks that can be dynamically added
type dynamicStageHook struct {
	preRun  map[string]func(context.Context, Step) error
	postRun map[string]func(context.Context, Step) error
	log     *logrus.Entry
}

func (d *dynamicStageHook) AddPreHook(s string, f func(context.Context, Step) error) {
	d.preRun[s] = f
}

func (d *dynamicStageHook) AddPostHook(s string, f func(context.Context, Step) error) {
	d.postRun[s] = f
}
func (d *dynamicStageHook) PreRun(ctx context.Context, s Step) error {
	for k, v := range d.preRun {
		if k == s.String() {
			d.log.Warnf("Running %s pre-run %s", s.String(), friendlyName(v))
			err := v(ctx, s)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *dynamicStageHook) PostRun(ctx context.Context, s Step) error {
	for k, v := range d.postRun {
		if k == s.String() {
			d.log.Warnf("Running %s post-run %s", s.String(), friendlyName(v))
			err := v(ctx, s)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func NewDynamicStageHook(log *logrus.Entry) *dynamicStageHook {
	return &dynamicStageHook{
		preRun:  make(map[string]func(context.Context, Step) error),
		postRun: make(map[string]func(context.Context, Step) error),
		log:     log,
	}
}
