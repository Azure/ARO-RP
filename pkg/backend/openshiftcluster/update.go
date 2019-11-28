package openshiftcluster

import (
	"context"
	"reflect"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
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

func (b *Manager) Update(ctx context.Context) error {
	machinesets, err := b.machinesets.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	have := 0
	for _, machineset := range machinesets.Items {
		if machineset.Spec.Replicas == nil {
			machineset.Spec.Replicas = to.Int32Ptr(1)
		}
		have += int(*machineset.Spec.Replicas)
	}

	for have > b.oc.Properties.WorkerProfiles[0].Count {
		machineset := find(machinesets.Items, func(i, j int) bool { return *machinesets.Items[i].Spec.Replicas > *machinesets.Items[j].Spec.Replicas }).(*machinev1beta1.MachineSet)
		*machineset.Spec.Replicas--
		have--
	}

	for have < b.oc.Properties.WorkerProfiles[0].Count {
		machineset := find(machinesets.Items, func(i, j int) bool { return *machinesets.Items[i].Spec.Replicas < *machinesets.Items[j].Spec.Replicas }).(*machinev1beta1.MachineSet)
		*machineset.Spec.Replicas++
		have++
	}

	for _, machineset := range machinesets.Items {
		want := *machineset.Spec.Replicas

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			machineset, err := b.machinesets.Get(machineset.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			b.log.Printf("scaling machineset %s to %d replicas", machineset.Name, want)
			machineset.Spec.Replicas = to.Int32Ptr(want)
			_, err = b.machinesets.Update(machineset)
			return err
		})
		if err != nil {
			return err
		}
	}

	for _, machineset := range machinesets.Items {
		want := *machineset.Spec.Replicas

		b.log.Printf("waiting for machineset %s", machineset.Name)
		err := wait.PollImmediate(10*time.Second, 30*time.Minute, func() (bool, error) {
			m, err := b.machinesets.Get(machineset.Name, metav1.GetOptions{})
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
