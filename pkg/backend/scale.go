package backend

import (
	"context"
	"reflect"
	"time"

	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	clusterapiclient "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/restconfig"
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

func (b *backend) scale(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster) error {
	restConfig, err := restconfig.RestConfig(oc.Properties.AdminKubeconfig)
	if err != nil {
		return err
	}

	cli, err := clusterapiclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	l, err := cli.MachineV1beta1().MachineSets("openshift-machine-api").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	have := 0
	for _, m := range l.Items {
		if m.Spec.Replicas == nil {
			m.Spec.Replicas = &[]int32{1}[0]
		}
		have += int(*m.Spec.Replicas)
	}

	for have > oc.Properties.WorkerProfiles[0].Count {
		m := find(l.Items, func(i, j int) bool { return *l.Items[i].Spec.Replicas > *l.Items[j].Spec.Replicas }).(*machinev1beta1.MachineSet)
		*m.Spec.Replicas--
		have--
	}

	for have < oc.Properties.WorkerProfiles[0].Count {
		m := find(l.Items, func(i, j int) bool { return *l.Items[i].Spec.Replicas < *l.Items[j].Spec.Replicas }).(*machinev1beta1.MachineSet)
		*m.Spec.Replicas++
		have++
	}

	for _, m := range l.Items {
		want := *m.Spec.Replicas

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			m, err := cli.MachineV1beta1().MachineSets(m.Namespace).Get(m.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			log.Printf("scaling machineset %s to %d replicas", m.Name, want)
			m.Spec.Replicas = &[]int32{want}[0]
			_, err = cli.MachineV1beta1().MachineSets(m.Namespace).Update(m)
			return err
		})
		if err != nil {
			return err
		}
	}

	for _, m := range l.Items {
		want := *m.Spec.Replicas

		log.Printf("waiting for machineset %s", m.Name)
		err := wait.PollImmediate(10*time.Second, 30*time.Minute, func() (bool, error) {
			m, err := cli.MachineV1beta1().MachineSets(m.Namespace).Get(m.Name, metav1.GetOptions{})
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
