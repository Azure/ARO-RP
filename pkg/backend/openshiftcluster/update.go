package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	machine "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset/typed/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

func find(xs interface{}, f func(int, int) bool) interface{} {
	v := reflect.ValueOf(xs)
	j := 0
	for i := 0; i < v.Len(); i++ {
		if f(i, j) {
			j = i
		}
	}
	return v.Index(j).Addr().Interface()
}

func (m *Manager) Update(ctx context.Context) error {
	ip, err := m.privateendpoint.GetIP(ctx, m.doc)
	if err != nil {
		return err
	}

	restConfig, err := restconfig.RestConfig(ctx, m.env, m.doc, ip)
	if err != nil {
		return err
	}

	cli, err := machine.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	machinesets, err := cli.MachineSets("openshift-machine-api").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	{
		items := make([]machinev1beta1.MachineSet, 0, len(machinesets.Items))
		for _, machineset := range machinesets.Items {
			if strings.HasPrefix(machineset.Name, "aro-worker-") {
				items = append(items, machineset)
			}
		}
		machinesets.Items = items
	}

	if len(machinesets.Items) == 0 {
		return fmt.Errorf("no worker machinesets found")
	}

	have := 0
	for _, machineset := range machinesets.Items {
		if machineset.Spec.Replicas == nil {
			machineset.Spec.Replicas = to.Int32Ptr(1)
		}
		have += int(*machineset.Spec.Replicas)
	}

	for have > m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].Count {
		machineset := find(machinesets.Items, func(i, j int) bool { return *machinesets.Items[i].Spec.Replicas > *machinesets.Items[j].Spec.Replicas }).(*machinev1beta1.MachineSet)
		*machineset.Spec.Replicas--
		have--
	}

	for have < m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].Count {
		machineset := find(machinesets.Items, func(i, j int) bool { return *machinesets.Items[i].Spec.Replicas < *machinesets.Items[j].Spec.Replicas }).(*machinev1beta1.MachineSet)
		*machineset.Spec.Replicas++
		have++
	}

	for _, machineset := range machinesets.Items {
		want := *machineset.Spec.Replicas

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			machineset, err := cli.MachineSets(machineset.Namespace).Get(machineset.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if machineset.Spec.Replicas == nil {
				machineset.Spec.Replicas = to.Int32Ptr(1)
			}

			if *machineset.Spec.Replicas != want {
				m.log.Printf("scaling machineset %s from %d to %d replicas", machineset.Name, *machineset.Spec.Replicas, want)
				machineset.Spec.Replicas = to.Int32Ptr(want)
				_, err = cli.MachineSets(machineset.Namespace).Update(machineset)
			}
			return err
		})
		if err != nil {
			return err
		}
	}

	for _, machineset := range machinesets.Items {
		want := *machineset.Spec.Replicas

		m.log.Printf("waiting for machineset %s", machineset.Name)
		err := wait.PollImmediate(10*time.Second, 30*time.Minute, func() (bool, error) {
			m, err := cli.MachineSets(machineset.Namespace).Get(machineset.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			return m.Status.ObservedGeneration == m.Generation &&
				m.Status.AvailableReplicas == want, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
