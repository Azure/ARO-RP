package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/openshift/installer/pkg/asset/password"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/restconfig"
)

func (i *Installer) removeBootstrap(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	g, err := i.getGraph(ctx, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	kubeadminPassword := g[reflect.TypeOf(&password.KubeadminPassword{})].(*password.KubeadminPassword)

	{
		i.log.Print("removing bootstrap vm")
		err := i.virtualmachines.DeleteAndWait(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap")
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap disk")
		err := i.disks.DeleteAndWait(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap_OSDisk")
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap nic")
		err = i.interfaces.DeleteAndWait(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap-nic")
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap ip")
		err = i.publicipaddresses.DeleteAndWait(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap-pip")
		if err != nil {
			return err
		}
	}

	{
		restConfig, err := restconfig.RestConfig(doc.OpenShiftCluster.Properties.AdminKubeconfig)
		if err != nil {
			return err
		}

		cli, err := configclient.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		i.log.Print("waiting for version clusterversion")
		now := time.Now()
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
	out:
		for {
			cv, err := cli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
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

	ips, err := i.publicipaddresses.List(ctx, doc.OpenShiftCluster.Properties.ResourceGroup)
	if err != nil {
		return err
	}

	{
		var routerIP string
		for _, ip := range ips {
			if ip.Tags["kubernetes-cluster-name"] != nil && *ip.Tags["kubernetes-cluster-name"] == "aro" &&
				ip.Tags["service"] != nil && *ip.Tags["service"] == "openshift-ingress/router-default" {
				routerIP = *ip.IPAddress
			}
		}
		if routerIP == "" {
			return fmt.Errorf("routerIP not found")
		}

		err = i.env.DNS().CreateOrUpdateRouter(ctx, doc.OpenShiftCluster, routerIP)
		if err != nil {
			return err
		}
	}

	_, err = i.db.Patch(doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerURL = "https://api." + doc.OpenShiftCluster.Properties.DomainName + "." + i.env.DNS().Domain() + ":6443/"
		doc.OpenShiftCluster.Properties.ConsoleURL = "https://console-openshift-console.apps." + doc.OpenShiftCluster.Properties.DomainName + "." + i.env.DNS().Domain() + "/"
		doc.OpenShiftCluster.Properties.KubeadminPassword = kubeadminPassword.Password
		return nil
	})
	return err
}
