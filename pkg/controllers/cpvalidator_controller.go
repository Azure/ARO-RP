package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

// CPValidator validate the cluster service principal
type CPValidator struct {
	Kubernetescli kubernetes.Interface
	AROCli        aroclient.AroV1alpha1Interface
	Log           *logrus.Entry
	Scheme        *runtime.Scheme
	sr            *StatusReporter
}

func (r *CPValidator) validateClusterServicePrincipal(ctx context.Context, instance *aro.Cluster) error {
	s, err := r.Kubernetescli.CoreV1().Secrets(OperatorNamespace).Get("service-principal", metav1.GetOptions{})
	if err != nil {
		return err
	}

	spp := &api.ServicePrincipalProfile{}
	err = json.Unmarshal(s.Data["servicePrincipal"], spp)
	if err != nil {
		return err
	}

	spnV := validate.NewServicePrincipalValidator(r.Log, spp, instance.Spec.ResourceID, instance.Spec.MasterSubnetID, instance.Spec.WorkerSubnetIDs[0])
	err = spnV.Validate(ctx)
	if err != nil {
		return r.sr.SetConditionFalse(ctx, aro.ClusterServicePrincipalAuthorized, err.Error())
	}
	return r.sr.SetConditionTrue(ctx, aro.ClusterServicePrincipalAuthorized, "can perfrom the required actions")
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// Reconcile will keep checking that the cluster has essential permissions.
func (r *CPValidator) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.Name != aro.SingletonClusterName {
		return reconcile.Result{}, nil
	}
	ctx := context.TODO()
	instance, err := r.AROCli.Clusters().Get(request.Name, v1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}
	if instance.Spec.ResourceID == "" {
		return ReconcileResultRequeueShort, nil
	}
	r.Log.Info("Period cluster checks")

	err = r.validateClusterServicePrincipal(ctx, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.Log.Info("done, requeueing")
	return ReconcileResultRequeueShort, nil
}

// SetupWithManager setup our mananger
func (r *CPValidator) SetupWithManager(mgr ctrl.Manager) error {
	r.sr = NewStatusReporter(r.Log, r.AROCli, aro.SingletonClusterName)

	return ctrl.NewControllerManagedBy(mgr).
		For(&aro.Cluster{}).
		Complete(r)
}
