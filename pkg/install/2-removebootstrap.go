package install

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/openshift/installer/pkg/asset/password"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/restconfig"
)

func (i *Installer) removeBootstrap(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	g, err := i.getGraph(ctx, doc)
	if err != nil {
		return err
	}

	kubeadminPassword := g[reflect.TypeOf(&password.KubeadminPassword{})].(*password.KubeadminPassword)

	{
		i.log.Print("removing bootstrap vm")
		future, err := i.virtualmachines.Delete(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, doc.OpenShiftCluster.Properties.ClusterID+"-bootstrap")
		if err != nil {
			return err
		}

		err = future.WaitForCompletionRef(ctx, i.virtualmachines.Client)
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap disk")
		future, err := i.disks.Delete(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, doc.OpenShiftCluster.Properties.ClusterID+"-bootstrap_OSDisk")
		if err != nil {
			return err
		}

		err = future.WaitForCompletionRef(ctx, i.disks.Client)
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap nic")
		future, err := i.interfaces.Delete(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, doc.OpenShiftCluster.Properties.ClusterID+"-bootstrap-nic")
		if err != nil {
			return err
		}

		err = future.WaitForCompletionRef(ctx, i.interfaces.Client)
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap ip")
		future, err := i.publicipaddresses.Delete(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, doc.OpenShiftCluster.Properties.ClusterID+"-bootstrap-pip")
		if err != nil {
			return err
		}

		err = future.WaitForCompletionRef(ctx, i.publicipaddresses.Client)
		if err != nil {
			return err
		}
	}

	{
		restConfig, err := restconfig.RestConfig(doc.OpenShiftCluster.Properties.AdminKubeconfig)
		if err != nil {
			return err
		}

		cli, err := configv1client.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		i.log.Print("waiting for version clusterversion")
		now := time.Now()
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
	out:
		for {
			cv, err := cli.ClusterVersions().Get("version", metav1.GetOptions{})
			if err == nil {
				for _, cond := range cv.Status.Conditions {
					if cond.Type == configv1.OperatorAvailable && cond.Status == configv1.ConditionTrue {
						break out
					}
				}
			}

			if time.Now().Sub(now) > 30*time.Minute {
				return fmt.Errorf("timed out waiting for version clusterversion")
			}

			<-t.C
		}
	}

	doc, err = i.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerURL = "https://api." + doc.OpenShiftCluster.Name + "." + os.Getenv("DOMAIN") + ":6443/"
		doc.OpenShiftCluster.Properties.ConsoleURL = "https://console-openshift-console.apps." + doc.OpenShiftCluster.Name + "." + os.Getenv("DOMAIN") + "/"
		doc.OpenShiftCluster.Properties.KubeadminPassword = kubeadminPassword.Password
		return nil
	})
	return err
}
