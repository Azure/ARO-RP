package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestUpdateFieldsFromPath(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		wantFields logrus.Fields
	}{
		{
			name: "normal resource",
			path: "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			wantFields: logrus.Fields{
				"subscription_id": "subscriptionid",
				"resource_group":  "resourcegroup",
				"resource_name":   "resourcename",
				"resource_id":     "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			},
		},
		{
			name: "normal resource and subresource",
			path: "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/foo",
			wantFields: logrus.Fields{
				"subscription_id": "subscriptionid",
				"resource_group":  "resourcegroup",
				"resource_name":   "resourcename",
				"resource_id":     "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			},
		},
		{
			name: "list resources in resource group",
			path: "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters",
			wantFields: logrus.Fields{
				"subscription_id": "subscriptionid",
				"resource_group":  "resourcegroup",
			},
		},
		{
			name: "list resources in subscription",
			path: "/subscriptions/subscriptionid/providers/microsoft.redhatopenshift/openshiftclusters",
			wantFields: logrus.Fields{
				"subscription_id": "subscriptionid",
			},
		},
		{
			name:       "non-resource",
			path:       "/healthz",
			wantFields: logrus.Fields{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fields := logrus.Fields{}

			updateFieldsFromPath(tt.path, fields)

			if !reflect.DeepEqual(fields, tt.wantFields) {
				t.Error(fields)
			}
		})
	}
}
