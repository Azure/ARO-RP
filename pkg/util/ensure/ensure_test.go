package ensure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	projectv1 "github.com/openshift/api/project/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned/fake"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNamespace(t *testing.T) {
	kubecli := fake.NewSimpleClientset(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "privileged-ifreload",
		},
	})
	seccli := &securityclient.Clientset{}

	ensurer := &ensure{
		cli:    kubecli,
		seccli: seccli,
	}
	err := ensurer.Namespace("privileged-ifreload")
	if err != nil {
		t.Fatal("could not ensure namespace", err)
	}

	ns, err := kubecli.CoreV1().Namespaces().Get("privileged-ifreload", metav1.GetOptions{})

	if err != nil {
		t.Fatal("could not get namespace", err)
	}

	if ns.Annotations == nil || ns.Annotations[projectv1.ProjectNodeSelector] != "" {
		t.Fatal("Annotation was not added to project", ns.Annotations)
	}
}
