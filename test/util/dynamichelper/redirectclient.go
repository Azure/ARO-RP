package dynamichelper

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

type redirectingClient struct {
	f client.WithWatch

	getHook    func(key client.ObjectKey, obj client.Object) error
	deleteHook func(obj client.Object) error
	createHook func(obj client.Object) error
	updateHook func(obj client.Object) error
}

var _ client.Client = &redirectingClient{}

func NewRedirectingClient(c client.WithWatch) *redirectingClient {
	return &redirectingClient{
		f: c,
	}
}

func (c *redirectingClient) WithGetHook(f func(key client.ObjectKey, obj client.Object) error) *redirectingClient {
	c.getHook = f
	return c
}

func (c *redirectingClient) WithDeleteHook(f func(obj client.Object) error) *redirectingClient {
	c.deleteHook = f
	return c
}

func (c *redirectingClient) WithCreateHook(f func(obj client.Object) error) *redirectingClient {
	c.createHook = f
	return c
}
func (c *redirectingClient) WithUpdateHook(f func(obj client.Object) error) *redirectingClient {
	c.updateHook = f
	return c
}

func (c *redirectingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	if c.getHook != nil {
		err := c.getHook(key, obj)
		if err != nil {
			return err
		}
	}

	return c.f.Get(ctx, key, obj)
}

func (c *redirectingClient) Watch(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	return c.f.Watch(ctx, list, opts...)
}

func (c *redirectingClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	return c.f.List(ctx, obj, opts...)
}

func (c *redirectingClient) Scheme() *runtime.Scheme {
	return c.f.Scheme()
}

func (c *redirectingClient) RESTMapper() meta.RESTMapper {
	return c.f.RESTMapper()
}

func (c *redirectingClient) Status() client.StatusWriter {
	return c.f.Status()
}

func (c *redirectingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.createHook != nil {
		err := c.createHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Create(ctx, obj, opts...)
}

func (c *redirectingClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.deleteHook != nil {
		err := c.deleteHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Delete(ctx, obj, opts...)
}
func (c *redirectingClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.f.DeleteAllOf(ctx, obj, opts...)
}

func (c *redirectingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.updateHook != nil {
		err := c.updateHook(obj)
		if err != nil {
			return err
		}
	}
	return c.f.Update(ctx, obj, opts...)
}
func (c *redirectingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
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
