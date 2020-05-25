package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func unmarshal(b []byte) (unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	err := yaml.Unmarshal(b, obj)
	return *obj, err
}

func TestClean(t *testing.T) {
	matches, err := filepath.Glob("testdata/clean/*-in.yaml")
	if err != nil {
		t.Fatal(err)
	}

	for _, match := range matches {
		b, err := ioutil.ReadFile(match)
		if err != nil {
			t.Error(err)
		}
		i, err := unmarshal(b)
		if err != nil {
			t.Error(err)
		}

		b, err = ioutil.ReadFile(strings.Replace(match, "-in.yaml", "-out.yaml", -1))
		if err != nil {
			t.Error(err)
		}
		o, err := unmarshal(b)
		if err != nil {
			t.Error(err)
		}

		clean(i)
		if !reflect.DeepEqual(i, o) {
			t.Errorf("%s:\n%s", match, cmp.Diff(i, o))
		}
	}
}
