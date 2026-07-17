package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

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
		vers, _, err := scheme.Scheme.ObjectKinds(obj)
		if err != nil {
			return nil
		}

		tally[vers[0].Kind] += 1
		return nil
	}
}

// TallyCountsAndKey will update tally with the Kubernetes Kind, object
// namespace, and object name (separated by '/') that pass through this hook.
func TallyCountsAndKey(tally map[string]int) hookFunc {
	return func(obj client.Object) error {
		vers, _, err := scheme.Scheme.ObjectKinds(obj)
		if err != nil {
			return nil
		}

		key := vers[0].Kind + "/" + types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}.String()
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

	diff := cmp.Diff(expected, actual, cmpopts.EquateEmpty())
	if diff == "" {
		return nil, nil
	}

	r := strings.Split(diff, "\n")
	if len(r) != 0 {
		return r, errors.New("tallies are not equal")
	} else {
		return nil, nil
	}
}

func copyForComparison(inObj any) (client.Object, error) {
	runtimeObj, ok := inObj.(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("cannot convert %v to runtime.Object", inObj)
	}

	ourObj, ok := runtimeObj.DeepCopyObject().(client.Object)
	if !ok {
		return nil, fmt.Errorf("cannot convert %v to runtime.Object", runtimeObj)
	}

	// Don't test for the resourceversion
	ourObj.SetResourceVersion("")
	// Don't test for the typemeta
	ourObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})

	return ourObj, nil
}

// Compare two objects. Calls t.Error() with a diff if they do not match.
func compareObjects(got, want any) ([]string, error) {
	ourGot, err := copyForComparison(got)
	if err != nil {
		return nil, err
	}
	ourWant, err := copyForComparison(want)
	if err != nil {
		return nil, err
	}

	r := strings.TrimSpace(cmp.Diff(ourGot, ourWant, cmpopts.EquateEmpty()))
	if r == "" {
		return []string{}, nil
	}

	return strings.Split(r, "\n"), nil
}

func CompareObjects(t *testing.T, got, want runtime.Object) {
	diff, err := compareObjects(got, want)
	if err != nil {
		t.Error(err)
		return
	}
	for _, i := range diff {
		if i != "" {
			t.Error(i)
		}
	}
}

// Compare two objects. Calls t.Error() with a diff if they do not match.
func CompareObjectList(t *testing.T, got, want []runtime.Object) {
	var gotList, wantList []client.Object

	for _, i := range got {
		obj, err := copyForComparison(i)
		if err != nil {
			t.Error(err)
			return
		}
		gotList = append(gotList, obj)
	}
	for _, i := range want {
		obj, err := copyForComparison(i)
		if err != nil {
			t.Error(err)
			return
		}
		wantList = append(wantList, obj)
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
	diff, err := compareObjects(actual, m.Expected)
	return len(diff) == 0, err
}

func (m *beEqualKubernetesObjects) FailureMessage(actual any) (message string) {
	diff, err := compareObjects(actual, m.Expected)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return "expected objects to equal:\n" + strings.Join(diff, "\n")
}

func (m *beEqualKubernetesObjects) NegatedFailureMessage(actual any) (message string) {
	diff, err := compareObjects(actual, m.Expected)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return "expected objects to be different:\n" + strings.Join(diff, "\n")
}

func BeEqualToKubernetesObject(obj runtime.Object) *beEqualKubernetesObjects {
	return &beEqualKubernetesObjects{Expected: obj}
}

// Create a new fake client which automatically adds arov1alpha1.Cluster{} as
// something with a status subresource.
func NewAROFakeClientBuilder(objects ...client.Object) *ctrlfake.ClientBuilder {
	return ctrlfake.NewClientBuilder().WithObjects(objects...).WithStatusSubresource(&arov1alpha1.Cluster{}, &configv1.ClusterOperator{})
}
