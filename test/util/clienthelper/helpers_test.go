package clienthelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
)

func TestTallyCountsAndKey(t *testing.T) {
	objs := []client.Object{
		&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somedep",
				Namespace: "somens",
			},
		},
	}

	tally := make(map[string]int)
	hook := TallyCountsAndKey(tally)

	for _, obj := range objs {
		err := hook(obj)
		require.NoError(t, err)
	}

	require.Equal(t, map[string]int{
		"ClusterVersion//version":   1,
		"Deployment/somens/somedep": 1,
	}, tally)
}

func TestTallyCountsAndKeyFailObjectKind(t *testing.T) {
	// We do not register the oauth APIs in the scheme so this should fail
	obj := &oauthv1.OAuthAccessToken{}

	tally := make(map[string]int)
	hook := TallyCountsAndKey(tally)

	err := hook(obj)
	require.ErrorContains(t, err, "when looking up objectkinds: no kind is registered for the type v1.OAuthAccessToken")

	// Should not be added to the tally
	require.Equal(t, map[string]int{}, tally)
}
