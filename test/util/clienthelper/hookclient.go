package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type getFunc func(key client.ObjectKey, obj client.Object) error
type hookFunc func(obj client.Object) error

type HookingClient struct {
	f client.WithWatch

	getHook    []getFunc
	deleteHook []hookFunc
	createHook []hookFunc
	updateHook []hookFunc
	patchHook  []hookFunc
}

var _ client.Client = &HookingClient{}

// NewHookingClient creates a
func NewHookingClient(c client.WithWatch) *HookingClient {
	return &HookingClient{
		f:          c,
		getHook:    []getFunc{},
		deleteHook: []hookFunc{},
		createHook: []hookFunc{},
		updateHook: []hookFunc{},
		patchHook:  []hookFunc{},
	}
}

func (c *HookingClient) WithGetHook(f getFunc) *HookingClient {
	c.getHook = append(c.getHook, f)
	return c
}

func (c *HookingClient) WithDeleteHook(f hookFunc) *HookingClient {
	c.deleteHook = append(c.deleteHook, f)
	return c
}

func (c *HookingClient) WithCreateHook(f hookFunc) *HookingClient {
	c.createHook = append(c.createHook, f)
	return c
}
func (c *HookingClient) WithUpdateHook(f hookFunc) *HookingClient {
	c.updateHook = append(c.updateHook, f)
	return c
}
func (c *HookingClient) WithPatchHook(f hookFunc) *HookingClient {
	c.patchHook = append(c.patchHook, f)
	return c
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Reader.Get]
func (c *HookingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	for _, h := range c.getHook {
		err := h(key, obj)
		if err != nil {
			return err
		}
	}
	return c.f.Get(ctx, key, obj)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Reader.List]
func (c *HookingClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	return c.f.List(ctx, obj, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.WithWatch.Watch]
func (c *HookingClient) Watch(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	return c.f.Watch(ctx, list, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Create]
func (c *HookingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	for _, h := range c.createHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Create(ctx, obj, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Delete]
func (c *HookingClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	for _, h := range c.deleteHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Delete(ctx, obj, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.DeleteAllOf]
func (c *HookingClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.f.DeleteAllOf(ctx, obj, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Update]
func (c *HookingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	for _, h := range c.updateHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Update(ctx, obj, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Patch]
func (c *HookingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	for _, h := range c.patchHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Patch(ctx, obj, patch, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Client.Scheme]
func (c *HookingClient) Scheme() *runtime.Scheme {
	return c.f.Scheme()
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Client.RESTMapper]
func (c *HookingClient) RESTMapper() meta.RESTMapper {
	return c.f.RESTMapper()
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Client.Status]
func (c *HookingClient) Status() client.StatusWriter {
	return c.f.Status()
}
