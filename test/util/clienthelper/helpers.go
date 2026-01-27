package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp/cmpopts"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

// TallyCounts will update tally with the Kubernetes Kinds that pass through
// this hook.
func TallyCounts(tally map[string]int) hookFunc {
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

// TallyCountsAndKey will update tally with the Kubernetes Kind, object
// namespace, and object name (separated by '/') that pass through this hook.
func TallyCountsAndKey(tally map[string]int) hookFunc {
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

// CompareTally will compare the two given tallies and return the differences
// and an error if they are different.
func CompareTally(expected map[string]int, actual map[string]int) ([]string, error) {
	// If both are empty or nil, they match
	if len(actual) == 0 && len(expected) == 0 {
		return nil, nil
	}

	// Make sure they're not nil maps before trying to sort + compare them
	if expected == nil {
		expected = map[string]int{}
	}
	if actual == nil {
		actual = map[string]int{}
	}

	r := deep.Equal(expected, actual)
	if len(r) != 0 {
		return r, errors.New("tallies are not equal")
	} else {
		return nil, nil
	}
}

// Compare two objects. Calls t.Error() with a diff if they do not match.
func CompareObjects(t *testing.T, got, want runtime.Object) {
	ourGot := got.DeepCopyObject().(client.Object)
	ourWant := want.DeepCopyObject().(client.Object)

	// Don't test for the resourceversion
	ourGot.SetResourceVersion("")
	ourWant.SetResourceVersion("")
	// Don't test for the typemeta
	ourGot.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})
	ourWant.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})

	diff := strings.SplitSeq(cmp.Diff(ourGot, ourWant, cmpopts.EquateEmpty()), "\n")
	for i := range diff {
		if i != "" {
			t.Error(i)
		}
	}
}

// Create a new fake client which automatically adds arov1alpha1.Cluster{} as
// something with a status subresource.
func NewAROFakeClientBuilder(objects ...client.Object) *ctrlfake.ClientBuilder {
	return ctrlfake.NewClientBuilder().WithObjects(objects...).WithStatusSubresource(&arov1alpha1.Cluster{})
}
