package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestEnrichWithPath(t *testing.T) {
	for _, tt := range []struct {
		name     string
		path     string
		wantData logrus.Fields
	}{
		{
			name: "normal resource",
			path: "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			wantData: logrus.Fields{
				"subscription_id": "subscriptionid",
				"resource_group":  "resourcegroup",
				"resource_name":   "resourcename",
				"resource_id":     "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			},
		},
		{
			name: "normal resource and subresource",
			path: "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/foo",
			wantData: logrus.Fields{
				"subscription_id": "subscriptionid",
				"resource_group":  "resourcegroup",
				"resource_name":   "resourcename",
				"resource_id":     "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			},
		},
		{
			name: "list resources in resource group",
			path: "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters",
			wantData: logrus.Fields{
				"subscription_id": "subscriptionid",
				"resource_group":  "resourcegroup",
			},
		},
		{
			name: "list resources in subscription",
			path: "/subscriptions/subscriptionid/providers/microsoft.redhatopenshift/openshiftclusters",
			wantData: logrus.Fields{
				"subscription_id": "subscriptionid",
			},
		},
		{
			name: "non-resource",
			path: "/healthz",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := &logrus.Entry{}

			log = EnrichWithPath(log, tt.path)

			if !reflect.DeepEqual(log.Data, tt.wantData) {
				t.Error(log.Data)
			}
		})
	}
}

func TestRelativeFilePathPrettier(t *testing.T) {
	pc := make([]uintptr, 1)
	runtime.Callers(1, pc)
	currentFrames := runtime.CallersFrames(pc)
	currentFunc, _ := currentFrames.Next()
	currentFunc.Line = 11 // so it's not too fragile

	tests := []struct {
		name         string
		f            *runtime.Frame
		wantFunction string
		wantFile     string
	}{
		{
			name:         "current function",
			f:            &currentFunc,
			wantFunction: "log.TestRelativeFilePathPrettier()",
			wantFile:     "pkg/util/log/log_test.go:11",
		},
		{
			name:         "empty",
			f:            &runtime.Frame{},
			wantFunction: "()",
			wantFile:     ":0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function, file := relativeFilePathPrettier(tt.f)
			if function != tt.wantFunction {
				t.Error(function)
			}
			if file != tt.wantFile {
				t.Error(file)
			}
		})
	}
}
