package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type getFunc func(key client.ObjectKey, obj client.Object) error
type hookFunc func(obj client.Object) error

type HookingClient struct {
	f client.WithWatch

	preGetHook    []getFunc
	preDeleteHook []hookFunc
	preCreateHook []hookFunc
	preUpdateHook []hookFunc
	prePatchHook  []hookFunc

	postGetHook    []getFunc
	postDeleteHook []hookFunc
	postCreateHook []hookFunc
	postUpdateHook []hookFunc
	postPatchHook  []hookFunc
}

type HookingSubResourceClient struct {
}

var _ client.Client = &HookingClient{}

// NewHookingClient creates a client which allows hooks to be added before and
// after the client event. Errors returned in the hooks are returned to the
// caller directly, to simulate issues such as disconnections or errors.
// Prehooks that cause errors will cause the underlying wrapped client to not be
// called.
func NewHookingClient(c client.WithWatch) *HookingClient {
	return &HookingClient{
		f:             c,
		preGetHook:    []getFunc{},
		preDeleteHook: []hookFunc{},
		preCreateHook: []hookFunc{},
		preUpdateHook: []hookFunc{},
		prePatchHook:  []hookFunc{},

		postGetHook:    []getFunc{},
		postDeleteHook: []hookFunc{},
		postCreateHook: []hookFunc{},
		postUpdateHook: []hookFunc{},
		postPatchHook:  []hookFunc{},
	}
}

func (c *HookingClient) WithPostGetHook(f getFunc) *HookingClient {
	c.postGetHook = append(c.postGetHook, f)
	return c
}

func (c *HookingClient) WithPostDeleteHook(f hookFunc) *HookingClient {
	c.postDeleteHook = append(c.postDeleteHook, f)
	return c
}

func (c *HookingClient) WithPostCreateHook(f hookFunc) *HookingClient {
	c.postCreateHook = append(c.postCreateHook, f)
	return c
}

func (c *HookingClient) WithPostUpdateHook(f hookFunc) *HookingClient {
	c.postUpdateHook = append(c.postUpdateHook, f)
	return c
}

func (c *HookingClient) WithPostPatchHook(f hookFunc) *HookingClient {
	c.postPatchHook = append(c.postPatchHook, f)
	return c
}

func (c *HookingClient) WithPreGetHook(f getFunc) *HookingClient {
	c.preGetHook = append(c.preGetHook, f)
	return c
}

func (c *HookingClient) WithPreDeleteHook(f hookFunc) *HookingClient {
	c.preDeleteHook = append(c.preDeleteHook, f)
	return c
}

func (c *HookingClient) WithPreCreateHook(f hookFunc) *HookingClient {
	c.preCreateHook = append(c.preCreateHook, f)
	return c
}
func (c *HookingClient) WithPreUpdateHook(f hookFunc) *HookingClient {
	c.preUpdateHook = append(c.preUpdateHook, f)
	return c
}

func (c *HookingClient) WithPrePatchHook(f hookFunc) *HookingClient {
	c.prePatchHook = append(c.prePatchHook, f)
	return c
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Reader.Get]
func (c *HookingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	for _, h := range c.preGetHook {
		err := h(key, obj)
		if err != nil {
			return err
		}
	}

	err := c.f.Get(ctx, key, obj)
	if err != nil {
		return err
	}

	for _, h := range c.postGetHook {
		err := h(key, obj)
		if err != nil {
			return err
		}
	}
	return nil
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
	for _, h := range c.preCreateHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}

	err := c.f.Create(ctx, obj, opts...)
	if err != nil {
		return err
	}

	for _, h := range c.postCreateHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *HookingClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return c.f.GroupVersionKindFor(obj)
}

func (c *HookingClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.f.IsObjectNamespaced(obj)
}

func (c *HookingClient) SubResource(input string) client.SubResourceClient {
	return c.f.SubResource(input)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Delete]
func (c *HookingClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	for _, h := range c.preDeleteHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}

	err := c.f.Delete(ctx, obj, opts...)
	if err != nil {
		return err
	}

	for _, h := range c.postDeleteHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.DeleteAllOf]
func (c *HookingClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.f.DeleteAllOf(ctx, obj, opts...)
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Update]
func (c *HookingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	for _, h := range c.preUpdateHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}

	err := c.f.Update(ctx, obj, opts...)
	if err != nil {
		return err
	}

	for _, h := range c.postUpdateHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// See [sigs.k8s.io/controller-runtime/pkg/client.Writer.Patch]
func (c *HookingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	for _, h := range c.prePatchHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}

	err := c.f.Patch(ctx, obj, patch, opts...)
	if err != nil {
		return err
	}

	for _, h := range c.postPatchHook {
		err := h(obj)
		if err != nil {
			return err
		}
	}
	return nil
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
