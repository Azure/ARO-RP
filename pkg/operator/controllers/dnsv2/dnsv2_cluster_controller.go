package dnsv2

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/openshift/installer/pkg/aro/dnsv2"
)

const (
	ClusterControllerName = "Dnsv2Cluster"
	controllerEnabled     = "aro.dnsv2.enabled"
)

type ClusterReconciler struct {
	log *logrus.Entry

	dh dynamichelper.Interface

	client client.Client
}

func NewClusterReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *ClusterReconciler {
	return &ClusterReconciler{
		log:    log,
		dh:     dh,
		client: client,
	}
}

// Reconcile watches the ARO object, and if it changes, reconciles all the
// 99-%s-aro-dns machineconfigs
func (r *ClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	mcps := &mcv1.MachineConfigPoolList{}
	err = r.client.List(ctx, mcps)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, mcps.Items...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Named(ClusterControllerName).
		Complete(r)
}

func reconcileMachineConfigs(ctx context.Context, instance *arov1alpha1.Cluster, dh dynamichelper.Interface, mcps ...mcv1.MachineConfigPool) error {
	var resources []kruntime.Object
	for _, mcp := range mcps {
		resource, err := generateMachineConfig(instance.Spec.Domain, instance.Spec.APIIntIP, instance.Spec.IngressIP, mcp.Name, instance.Spec.GatewayDomains, instance.Spec.GatewayPrivateEndpointIP)
		if err != nil {
			return err
		}

		err = dynamichelper.SetControllerReferences([]kruntime.Object{resource}, &mcp)
		if err != nil {
			return err
		}

		resources = append(resources, resource)
	}

	err := dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	return dh.Ensure(ctx, resources...)
}

func generateMachineConfig(clusterDomain, apiIntIP, ingressIP, role string, gatewayDomains []string, gatewayPrivateEndpointIP string) (*mcv1.MachineConfig, error) {
	ignConfig, err := dnsv2.Ignition2Config(clusterDomain, apiIntIP, ingressIP, gatewayDomains, gatewayPrivateEndpointIP)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(ignConfig)
	if err != nil {
		return nil, err
	}

	// canonicalise the machineconfig payload the same way as MCO
	var i interface{}
	err = json.Unmarshal(b, &i)
	if err != nil {
		return nil, err
	}

	rawExt := kruntime.RawExtension{}
	rawExt.Raw, err = json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return &mcv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-aro-dns", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcv1.MachineConfigSpec{
			Config: rawExt,
		},
	}, nil
}
