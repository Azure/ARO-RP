package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	v1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDisableUpdates(t *testing.T) {
	ctx := context.Background()

	versionName := "version"

	m := &manager{
		configcli: fake.NewSimpleClientset(&v1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: versionName,
			},
			Spec: v1.ClusterVersionSpec{
				Upstream: "RemoveMe",
				Channel:  "RemoveMe",
			},
		}),
	}

	err := m.disableUpdates(ctx)
	if err != nil {
		t.Error(err)
	}

	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, versionName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if cv.Spec.Upstream != "" {
		t.Error(cv.Spec.Upstream)
	}
	if cv.Spec.Channel != "" {
		t.Error(cv.Spec.Channel)
	}
}
