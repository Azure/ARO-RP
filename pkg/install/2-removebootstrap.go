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
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/password"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

func (i *Installer) removeBootstrap(ctx context.Context) error {
	g, err := i.getGraph(ctx)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	kubeadminPassword := g[reflect.TypeOf(&password.KubeadminPassword{})].(*password.KubeadminPassword)

	{
		i.log.Print("removing bootstrap vm")
		err := i.virtualmachines.DeleteAndWait(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap")
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap disk")
		err := i.disks.DeleteAndWait(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap_OSDisk")
		if err != nil {
			return err
		}
	}

	{
		i.log.Print("removing bootstrap nic")
		err = i.interfaces.DeleteAndWait(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "aro-bootstrap-nic")
		if err != nil {
			return err
		}
	}

	var restConfig *rest.Config
	{
		ip, err := i.privateendpoint.GetIP(ctx, i.doc)
		if err != nil {
			return err
		}

		restConfig, err = restconfig.RestConfig(ctx, i.env, i.doc, ip)
		if err != nil {
			return err
		}
	}

	{
		cli, err := configclient.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		i.log.Print("waiting for version clusterversion")
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
			cv, err := cli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
			if err == nil {
				for _, cond := range cv.Status.Conditions {
					if cond.Type == configv1.OperatorAvailable && cond.Status == configv1.ConditionTrue {
						return true, nil
					}
				}
			}
			return false, nil

		}, timeoutCtx.Done())
		if err != nil {
			return err
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			cv, err := cli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
			if err != nil {
				return err
			}

			cv.Spec.Upstream = ""
			cv.Spec.Channel = ""

			_, err = cli.ConfigV1().ClusterVersions().Update(cv)
			return err
		})
		if err != nil {
			return err
		}
	}

	var routerIP string
	{
		cli, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		svc, err := cli.CoreV1().Services("openshift-ingress").Get("router-default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			return fmt.Errorf("routerIP not found")
		}

		routerIP = svc.Status.LoadBalancer.Ingress[0].IP
	}

	err = i.dns.CreateOrUpdateRouter(ctx, i.doc.OpenShiftCluster, routerIP)
	if err != nil {
		return err
	}

	i.doc, err = i.db.Patch(i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.APIServerProfile.URL = "https://api." + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + ":6443/"
		doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = routerIP
		doc.OpenShiftCluster.Properties.ConsoleURL = "https://console-openshift-console.apps." + installConfig.Config.ObjectMeta.Name + "." + installConfig.Config.BaseDomain + "/"
		doc.OpenShiftCluster.Properties.KubeadminPassword = kubeadminPassword.Password
		return nil
	})
	return err
}
