package ifreload

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/env"
	mock_ensure "github.com/Azure/ARO-RP/pkg/util/mocks/ensure"
)

func TestIfReloadCreateOrUpdate(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()
	m := mock_ensure.NewMockInterface(controller)

	ifreload := &ifReload{
		env:     &env.Test{},
		log:     logrus.NewEntry(logrus.StandardLogger()),
		ensurer: m,
	}
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "privileged",
		},
	}
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ifreload",
			Namespace: kubeNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "ifreload"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "ifreload"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "ifreload",
							Image: ifreload.ifreloadImage(),
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("100m"),
									v1.ResourceMemory: resource.MustParse("200Mi"),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("100m"),
									v1.ResourceMemory: resource.MustParse("200Mi"),
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: to.BoolPtr(true),
							},
						},
					},
					HostNetwork: true,
					Tolerations: []v1.Toleration{
						{
							Effect:   v1.TaintEffectNoExecute,
							Operator: v1.TolerationOpExists,
						},
						{
							Effect:   v1.TaintEffectNoSchedule,
							Operator: v1.TolerationOpExists,
						},
					},
				},
			},
		},
	}
	m.EXPECT().Namespace("openshift-azure-ifreload").Return(nil)
	m.EXPECT().SccGet().Return(scc, nil)
	m.EXPECT().SccCreate(scc).Return(nil)
	m.EXPECT().DaemonSet(ds).Return(nil)

	err := ifreload.CreateOrUpdate(ctx)

	if err != nil {
		t.Fatal(err)
	}
}
