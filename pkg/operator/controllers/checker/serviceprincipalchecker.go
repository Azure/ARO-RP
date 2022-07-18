package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"

	operatorv1 "github.com/openshift/api/operator/v1"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

type ServicePrincipalChecker struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
	maocli        machineclient.Interface

	role string
}

func NewServicePrincipalChecker(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface, maocli machineclient.Interface, role string) *ServicePrincipalChecker {
	return &ServicePrincipalChecker{
		log:           log,
		arocli:        arocli,
		kubernetescli: kubernetescli,
		maocli:        maocli,
		role:          role,
	}
}

func (r *ServicePrincipalChecker) Name() string {
	return "ServicePrincipalChecker"
}

func (r *ServicePrincipalChecker) Check(ctx context.Context) error {
	cond := &operatorv1.OperatorCondition{
		Type:    arov1alpha1.ServicePrincipalValid,
		Status:  operatorv1.ConditionTrue,
		Message: "service principal is valid",
		Reason:  "CheckDone",
	}

	cluster, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	azEnv, err := azureclient.EnvironmentFromName(cluster.Spec.AZEnvironment)
	if err != nil {
		return err
	}

	azCred, err := clusterauthorizer.AzCredentials(ctx, r.kubernetescli)
	if err != nil {
		return err
	}

	_, err = aad.GetToken(ctx, r.log, string(azCred.ClientID), string(azCred.ClientSecret), string(azCred.TenantID), azEnv.ActiveDirectoryEndpoint, azEnv.ResourceManagerEndpoint)
	if err != nil {
		updateFailedCondition(cond, err)
	}

	spDynamic, err := dynamic.NewServicePrincipalValidator(r.log, &azEnv, dynamic.AuthorizerClusterServicePrincipal)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateServicePrincipal(ctx, string(azCred.ClientID), string(azCred.ClientSecret), string(azCred.TenantID), azureclaim.AzureClaim{}, dynamic.GetServicePrincipalToken)
	if err != nil {
		updateFailedCondition(cond, err)
	}

	return conditions.SetCondition(ctx, r.arocli, cond, r.role)
}

func updateFailedCondition(cond *operatorv1.OperatorCondition, err error) {
	cond.Status = operatorv1.ConditionFalse
	if tErr, ok := err.(*api.CloudError); ok {
		cond.Message = tErr.Message
	} else {
		cond.Message = err.Error()
	}
}
