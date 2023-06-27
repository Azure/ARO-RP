package cluster

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// This test file serves as documentation about an interesting behavior of Go,
// an error we faced in production consisting of
// evaluating mon.hiveclientset == nil when the interface is initialised from a function call
// that returned an error and later using the interface and geting a panic.

// TestEmitHiveRegistrationPanics shows that we get a panic when we can not connect to AKS due to us using a interface
// (hiveclientset) we obtained doing client.Get() which returned an error (but we did not care about the error).
// The problem is that hiveclientset is an interface with type *client.Client and value nil and hiveclientset == nil evaluates to false
// and later mon.hiveclientset.Get() panics.
func TestEmitHiveRegistrationPanics(t *testing.T) {
	// GIVEN a hiveclientset interface declared which has interface type: nil, interface value: nil
	var hiveclientset client.Client
	fmt.Printf("hiveclientset: interface type is %T, interface value is: %v\n", hiveclientset, hiveclientset)
	fmt.Printf("hiveclientset == nil ? %v\n\n", hiveclientset == nil)

	// AND GIVEN the same hiveclientset assigned the result of a call to client.New() which returned non nil error,
	// here the interface type gets changed from nil to *client.Client
	// making future comparison hiveclientset == nil evaulate to FALSE (counter intuitive) and at the same time
	// mon.hiveclientset.Get panicking
	hiveclientset, err := client.New(nil, client.Options{})
	if err == nil {
		t.Fatalf("we want to make sure err is not nil for this test")
	}
	fmt.Printf("hiveclientset: interface type is %T, interface value is: %v\n", hiveclientset, hiveclientset)
	fmt.Printf("hiveclientset == nil ? %v\n\n", hiveclientset == nil)

	mon := &Monitor{
		// we use a hiveclientset coming from an errored client.New() which returned *client.Client -> NOT SAFE
		hiveclientset: hiveclientset,
		oc: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: "something",
				},
			},
		},
		log: utillog.GetLogger(),
	}

	// mon.hiveclientset == nil evaluating to false would be what we would expect without taking into account
	// the underlaying interface TYPE and VALUE, but work is tricky in that regard.
	if mon.hiveclientset == nil {
		t.Fatalf("mon.hiveclientset == nil evaluates to true")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("We expect to panic due to interface hiveclientset having *client.Client as type and nil as value")
		}
	}()

	// THEN we see a panic when we call
	// emitHiveRegistrationStatus-> mon.hiveclientset == nil ? is false -> retrieveClusterDeployment()->
	// -> err := mon.hiveclientset.Get(...) -> panic
	_ = mon.emitHiveRegistrationStatus(context.Background())
}

// TestEmitHiveRegistrationDoesNotPanicWhenWeAssignLiteralNil shows that if we assign explicitly nil
// to a hiveclientset after geting an error from client.New(), the assertion hiveclientset == nil evaluates to
// true and we avoid future panics.
func TestEmitHiveRegistrationDoesNotPanicWhenWeAssignLiteralNil(t *testing.T) {
	// interface type: *client.Client, interface value: nil
	hiveclientset, err := client.New(&rest.Config{}, client.Options{})
	fmt.Printf("hiveclientset: interface type is %T, interface value is: %v\n", hiveclientset, hiveclientset)
	fmt.Printf("hiveclientset == nil ? %v\n\n", hiveclientset == nil)
	if err == nil {
		t.Fatalf("this test wants client.New() to return an error to mimic the behavior in production")
	}

	if err != nil {
		// This modifies the interface type to nil, making hiveclientset == nil evaluate to true.
		hiveclientset = nil
	}

	fmt.Printf("hiveclientset: interface type is %T, interface value is: %v\n", hiveclientset, hiveclientset)
	fmt.Printf("hiveclientset == nil ? %v\n\n", hiveclientset == nil)

	mon := &Monitor{
		hiveclientset: hiveclientset,
		oc: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: "something",
				},
			},
		},
		log: utillog.GetLogger(),
	}

	if mon.hiveclientset != nil {
		t.Fatalf("mon.hiveclientset should be nil")
	}

	// no panic
	_ = mon.emitHiveRegistrationStatus(context.Background())
}

// TestEmitHiveRegistrationDoesNotPanicWhenWeAssignLiteralNilUsingAuxFunction shows the exact same
// behavior as TestEmitHiveRegistrationDoesNotPanicWhenWeAssignLiteralNil but encapsulating the
// initialisation of hiveclientset in a function. The conclusion is the same, assign literal nil
// to the interface when we get an error: the difference is that in this test this assignment is
// coming from a function the prod code uses (so it is a good think) while the previous tests was done inline in the test.
func TestEmitHiveRegistrationDoesNotPanicWhenWeAssignLiteralNilUsingAuxFunction(t *testing.T) {
	hiveclientset, err := getHiveClientSet(&rest.Config{})
	if err == nil {
		t.Fatalf("we want to make sure err is not nil for this test")
	}

	mon := &Monitor{
		hiveclientset: hiveclientset,
		oc: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: "something",
				},
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "something",
				},
			},
		},
		log: utillog.GetLogger(),
	}

	fmt.Printf("mon.hiveclientset: interface type is %T, interface value is: %v\n", mon.hiveclientset, mon.hiveclientset)
	if mon.hiveclientset != nil {
		t.Fatalf("mon.hiveclientset should be nil")
	}

	// no panic
	_ = mon.emitHiveRegistrationStatus(context.Background())
}
