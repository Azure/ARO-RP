package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HookingClient struct {
	f client.WithWatch

	getHook    func(key client.ObjectKey, obj client.Object) error
	deleteHook func(obj client.Object) error
	createHook func(obj client.Object) error
	updateHook func(obj client.Object) error
	patchHook  func(obj client.Object) error
}

var _ client.Client = &HookingClient{}

func NewHookingClient(c client.WithWatch) *HookingClient {
	return &HookingClient{
		f: c,
	}
}

func (c *HookingClient) WithGetHook(f func(key client.ObjectKey, obj client.Object) error) *HookingClient {
	c.getHook = f
	return c
}

func (c *HookingClient) WithDeleteHook(f func(obj client.Object) error) *HookingClient {
	c.deleteHook = f
	return c
}

func (c *HookingClient) WithCreateHook(f func(obj client.Object) error) *HookingClient {
	c.createHook = f
	return c
}
func (c *HookingClient) WithUpdateHook(f func(obj client.Object) error) *HookingClient {
	c.updateHook = f
	return c
}
func (c *HookingClient) WithPatchHook(f func(obj client.Object) error) *HookingClient {
	c.patchHook = f
	return c
}

func (c *HookingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	if c.getHook != nil {
		err := c.getHook(key, obj)
		if err != nil {
			return err
		}
	}

	return c.f.Get(ctx, key, obj)
}

func (c *HookingClient) Watch(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	return c.f.Watch(ctx, list, opts...)
}

func (c *HookingClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	return c.f.List(ctx, obj, opts...)
}

func (c *HookingClient) Scheme() *runtime.Scheme {
	return c.f.Scheme()
}

func (c *HookingClient) RESTMapper() meta.RESTMapper {
	return c.f.RESTMapper()
}

func (c *HookingClient) Status() client.StatusWriter {
	return c.f.Status()
}

func (c *HookingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.createHook != nil {
		err := c.createHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Create(ctx, obj, opts...)
}

func (c *HookingClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.deleteHook != nil {
		err := c.deleteHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Delete(ctx, obj, opts...)
}
func (c *HookingClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.f.DeleteAllOf(ctx, obj, opts...)
}

func (c *HookingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.updateHook != nil {
		err := c.updateHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Update(ctx, obj, opts...)
}
func (c *HookingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.patchHook != nil {
		err := c.patchHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Patch(ctx, obj, patch, opts...)
}

func TallyCounts(tally map[string]int) func(obj client.Object) error {
	return func(obj client.Object) error {
		m := meta.NewAccessor()
		kind, err := m.Kind(obj)
		if err != nil {
			return err
		}

		tally[kind] += 1
		return nil
	}
}

func TallyCountsAndKey(tally map[string]int) func(obj client.Object) error {
	return func(obj client.Object) error {
		m := meta.NewAccessor()
		kind, err := m.Kind(obj)
		if err != nil {
			return err
		}

		key := kind + "/" + types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}.String()
		tally[key] += 1
		return nil
	}
}
