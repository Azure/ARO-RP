package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/onsi/gomega/format"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

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

func copyForComparison(inObj any) client.Object {
	ourObj := inObj.(runtime.Object).DeepCopyObject().(client.Object)

	// Don't test for the resourceversion
	ourObj.SetResourceVersion("")
	// Don't test for the typemeta
	ourObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})

	return ourObj
}

// Compare two objects. Calls t.Error() with a diff if they do not match.
func CompareObjects(t *testing.T, got, want runtime.Object) {
	ourGot, ourWant := copyForComparison(got), copyForComparison(want)

	diff := strings.SplitSeq(cmp.Diff(ourGot, ourWant, cmpopts.EquateEmpty()), "\n")
	for i := range diff {
		if i != "" {
			t.Error(i)
		}
	}
}

// Compare two objects. Calls t.Error() with a diff if they do not match.
func CompareObjectList(t *testing.T, got, want []runtime.Object) {
	var gotList, wantList []client.Object

	for _, i := range got {
		gotList = append(gotList, copyForComparison(i))
	}
	for _, i := range want {
		wantList = append(wantList, copyForComparison(i))
	}

	m := meta.NewAccessor()

	sortObjs := func(i, j client.Object) int {
		iNamespace, err := m.Namespace(i)
		if err != nil {
			return 0
		}
		iName, err := m.Name(i)
		if err != nil {
			return 0
		}
		jNamespace, err := m.Namespace(j)
		if err != nil {
			return 0
		}
		jName, err := m.Name(j)
		if err != nil {
			return 0
		}

		return strings.Compare(iNamespace+"/"+iName, jNamespace+"/"+jName)
	}
	slices.SortFunc(gotList, sortObjs)
	slices.SortFunc(wantList, sortObjs)

	diff := strings.SplitSeq(cmp.Diff(gotList, wantList, cmpopts.EquateEmpty()), "\n")
	for i := range diff {
		if i != "" {
			t.Error(i)
		}
	}
}

type beEqualKubernetesObjects struct {
	Expected any
}

func (m *beEqualKubernetesObjects) Match(actual any) (success bool, err error) {
	ourGot, ourWant := copyForComparison(actual), copyForComparison(m.Expected)

	return reflect.DeepEqual(ourGot, ourWant), nil
}

func (m *beEqualKubernetesObjects) FailureMessage(actual any) (message string) {
	ourGot, ourWant := copyForComparison(actual), copyForComparison(m.Expected)

	return "expected objects to equal:\n" + cmp.Diff(ourGot, ourWant, cmpopts.EquateEmpty())
}

func (m *beEqualKubernetesObjects) NegatedFailureMessage(actual any) (message string) {
	ourGot, ourWant := copyForComparison(actual), copyForComparison(m.Expected)

	return format.Message(ourGot, "not to equal", ourWant)
}

func BeEqualToKubernetesObject(obj runtime.Object) *beEqualKubernetesObjects {
	return &beEqualKubernetesObjects{Expected: obj}
}

// Create a new fake client which automatically adds arov1alpha1.Cluster{} as
// something with a status subresource.
func NewAROFakeClientBuilder(objects ...client.Object) *ctrlfake.ClientBuilder {
	return ctrlfake.NewClientBuilder().WithObjects(objects...).WithStatusSubresource(&arov1alpha1.Cluster{}, &configv1.ClusterOperator{})
}
