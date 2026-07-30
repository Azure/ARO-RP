package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func newTestVMClient(t *testing.T, transport transportFunc) VirtualMachinesClient {
	t.Helper()
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: transport,
			Retry:     policy.RetryOptions{MaxRetries: -1},
		},
	}
	c, err := NewVirtualMachinesClient("sub-id", &fakeazcore.TokenCredential{}, opts)
	if err != nil {
		t.Fatalf("NewVirtualMachinesClient: %v", err)
	}
	return c
}

// Test_virtualMachinesClient_GetWithInstanceView_setsExpandParam verifies that
// GetWithInstanceView sets the $expand=instanceView query parameter, which is
// what actually causes the service to populate InstanceView on the response.
func Test_virtualMachinesClient_GetWithInstanceView_setsExpandParam(t *testing.T) {
	var gotExpand string
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		gotExpand = req.URL.Query().Get("$expand")
		return httpResponse(req, http.StatusOK, `{"name":"vm1","location":"eastus"}`), nil
	})

	c := newTestVMClient(t, transport)
	vm, err := c.GetWithInstanceView(context.Background(), "rg", "vm1")
	if err != nil {
		t.Fatalf("GetWithInstanceView() unexpected error: %v", err)
	}
	if gotExpand != "instanceView" {
		t.Errorf("expected $expand=instanceView, got %q", gotExpand)
	}
	if vm.Name == nil || *vm.Name != "vm1" {
		t.Errorf("expected VM name %q, got %v", "vm1", vm.Name)
	}
}

// Test_virtualMachinesClient_Get_noExpandParam verifies that the plain Get
// does NOT request the InstanceView expansion, distinguishing it from
// GetWithInstanceView.
func Test_virtualMachinesClient_Get_noExpandParam(t *testing.T) {
	var gotExpand string
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		gotExpand = req.URL.Query().Get("$expand")
		return httpResponse(req, http.StatusOK, `{"name":"vm1","location":"eastus"}`), nil
	})

	c := newTestVMClient(t, transport)
	vm, err := c.Get(context.Background(), "rg", "vm1")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if gotExpand != "" {
		t.Errorf("expected no $expand param, got %q", gotExpand)
	}
	if vm.Name == nil || *vm.Name != "vm1" {
		t.Errorf("expected VM name %q, got %v", "vm1", vm.Name)
	}
}

// Test_virtualMachinesClient_GetWithInstanceView_error verifies that a
// non-2xx response is propagated as an error rather than a zero-value VM.
func Test_virtualMachinesClient_GetWithInstanceView_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusNotFound, `{"error":{"code":"ResourceNotFound"}}`), nil
	})

	c := newTestVMClient(t, transport)
	vm, err := c.GetWithInstanceView(context.Background(), "rg", "vm1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if vm.Name != nil {
		t.Errorf("expected zero-value VM on error, got %v", vm)
	}
}

func Test_virtualMachinesClient_CreateOrUpdateAndWait_success(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, `{"name":"vm1","location":"eastus"}`), nil
	})

	c := newTestVMClient(t, transport)
	err := c.CreateOrUpdateAndWait(context.Background(), "rg", "vm1", armcompute.VirtualMachine{
		Location: pointerutils.ToPtr("eastus"),
	})
	if err != nil {
		t.Errorf("CreateOrUpdateAndWait() unexpected error: %v", err)
	}
}

func Test_virtualMachinesClient_CreateOrUpdateAndWait_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusConflict, `{"error":{"code":"Conflict"}}`), nil
	})

	c := newTestVMClient(t, transport)
	err := c.CreateOrUpdateAndWait(context.Background(), "rg", "vm1", armcompute.VirtualMachine{
		Location: pointerutils.ToPtr("eastus"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_virtualMachinesClient_UpdateAndWait_success(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, `{"name":"vm1","location":"eastus"}`), nil
	})

	c := newTestVMClient(t, transport)
	err := c.UpdateAndWait(context.Background(), "rg", "vm1", armcompute.VirtualMachineUpdate{})
	if err != nil {
		t.Errorf("UpdateAndWait() unexpected error: %v", err)
	}
}

func Test_virtualMachinesClient_UpdateAndWait_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusConflict, `{"error":{"code":"Conflict"}}`), nil
	})

	c := newTestVMClient(t, transport)
	err := c.UpdateAndWait(context.Background(), "rg", "vm1", armcompute.VirtualMachineUpdate{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_virtualMachinesClient_DeallocateAndWait_success(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, ""), nil
	})

	c := newTestVMClient(t, transport)
	err := c.DeallocateAndWait(context.Background(), "rg", "vm1")
	if err != nil {
		t.Errorf("DeallocateAndWait() unexpected error: %v", err)
	}
}

func Test_virtualMachinesClient_DeallocateAndWait_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusConflict, `{"error":{"code":"Conflict"}}`), nil
	})

	c := newTestVMClient(t, transport)
	err := c.DeallocateAndWait(context.Background(), "rg", "vm1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_virtualMachinesClient_StartAndWait_success(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusOK, ""), nil
	})

	c := newTestVMClient(t, transport)
	err := c.StartAndWait(context.Background(), "rg", "vm1")
	if err != nil {
		t.Errorf("StartAndWait() unexpected error: %v", err)
	}
}

func Test_virtualMachinesClient_StartAndWait_error(t *testing.T) {
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(req, http.StatusConflict, `{"error":{"code":"Conflict"}}`), nil
	})

	c := newTestVMClient(t, transport)
	err := c.StartAndWait(context.Background(), "rg", "vm1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
