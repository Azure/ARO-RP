package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	arov1alpha "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/aad"
)

type DNSChecker struct {
	arocli        aroclient.Interface
	log           *logrus.Entry
	kubernetescli kubernetes.Interface
	clustercli    maoclient.Interface
	role          string
}

func NewDNSChecker(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface, clustercli maoclient.Interface, role string) *DNSChecker {
	return &DNSChecker{
		log:           log,
		arocli:        arocli,
		kubernetescli: kubernetescli,
		clustercli:    clustercli,
		role:          role,
	}
}

func (d *DNSChecker) Name() string {
	return "DNSChecker"
}

func (d *DNSChecker) Check(ctx context.Context) error {
	instance, err := d.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	//Get masterSubnet from master machine
	masterSubnet, err := masterSubnetId(ctx, d.clustercli, instance.Spec.VnetID)
	if err != nil {
		return err
	}
	//Get endpoints from operator
	azEnv, err := azure.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		return err
	}
	//Grab azure-credentials from secret
	credentials, err := azCredentials(ctx, d.kubernetescli)
	if err != nil {
		return err
	}
	//create service principal token from azure-credentials
	token, err := aad.GetToken(ctx, d.log, string(credentials.clientID), api.SecureString(credentials.clientSecret), string(credentials.tenantID), azEnv.ActiveDirectoryEndpoint, azEnv.ResourceManagerEndpoint)
	if err != nil {
		return err
	}
	//create refreshable authorizer from token
	authorizer, err := newAuthorizer(token)
	if err != nil {
		return err
	}
	resource, err := azure.ParseResourceID(instance.Spec.ResourceID)
	if err != nil {
		return err
	}
	checker, err := validate.NewValidator(d.log, &azEnv, *masterSubnet, nil, resource.SubscriptionID, authorizer)
	if err != nil {
		return err
	}

	var condition *status.Condition

	err = checker.ValidateVnetDNS(ctx)
	if err != nil {
		condition = &status.Condition{
			Type:    arov1alpha.DNSValid,
			Status:  corev1.ConditionFalse,
			Message: "Custom DNS Found on VNET",
			Reason:  "CheckFailed",
		}
	} else {
		condition = &status.Condition{
			Type:    arov1alpha.DNSValid,
			Status:  corev1.ConditionTrue,
			Message: "DNS Check successful",
			Reason:  "CheckDone",
		}
	}
	err = controllers.SetCondition(ctx, d.arocli, condition, d.role)
	if err != nil {
		return err
	}
	return nil
}
