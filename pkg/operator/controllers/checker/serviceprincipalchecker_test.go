package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	mock_aad "github.com/Azure/ARO-RP/pkg/util/mocks/aad"
	mock_dynamic "github.com/Azure/ARO-RP/pkg/util/mocks/dynamic"
)

func TestServicePrincipalValid(t *testing.T) {
	ctx := context.Background()

	var (
		name      = "azure-credentials"
		nameSpace = "kube-system"
		log       = logrus.NewEntry(logrus.StandardLogger())
	)

	for _, tt := range []struct {
		name                 string
		aroCluster           *arov1alpha1.Cluster
		azureSecretName      string
		azureSecretNameSpace string
		azureSecret          string
		secret               *corev1.Secret
		wantErr              string
	}{
		{
			name:    "fail: aro cluster resource doesn't exist",
			wantErr: `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "fail: azure-credential secret doesn't exist",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: azuretypes.PublicCloud.Name(),
				},
			},
			wantErr: `secrets "azure-credentials" not found`,
		},
		{
			name: "pass: token authentication",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: azuretypes.PublicCloud.Name(),
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_id":     []byte("my-client-id"),
					"azure_client_secret": []byte("my-client-secret"),
					"azure_tenant_id":     []byte("my-tenant.example.com"),
				},
			},
		},
		{
			name: "pass: dynamic token authentication",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					AZEnvironment: azuretypes.PublicCloud.Name(),
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: nameSpace,
				},
				Data: map[string][]byte{
					"azure_client_id":     []byte("my-client-id"),
					"azure_client_secret": []byte("my-client-secret"),
					"azure_tenant_id":     []byte("my-tenant.example.com"),
				},
			},
		},
	} {
		arocli := arofake.NewSimpleClientset()
		kubernetescli := fake.NewSimpleClientset()

		if tt.aroCluster != nil {
			arocli = arofake.NewSimpleClientset(tt.aroCluster)
		}
		if tt.secret != nil {
			kubernetescli = fake.NewSimpleClientset(tt.secret)
		}

		controller := gomock.NewController(t)
		aad := mock_aad.NewMockTokenClient(controller)
		dynamicController := gomock.NewController(t)
		dynamic := mock_dynamic.NewMockDynamic(dynamicController)

		if tt.secret != nil {
			aadCall := aad.EXPECT().GetToken(ctx,
				log,
				string(tt.secret.Data["azure_client_id"]),
				string(tt.secret.Data["azure_client_secret"]),
				string(tt.secret.Data["azure_tenant_id"]),
				"https://login.microsoftonline.com/",
				"https://management.azure.com/").MaxTimes(1).Return(nil, nil)

			dynamic.EXPECT().ValidateServicePrincipal(ctx,
				string(tt.secret.Data["azure_client_id"]),
				string(tt.secret.Data["azure_client_secret"]),
				string(tt.secret.Data["azure_tenant_id"]),
			).MaxTimes(1).After(aadCall).Return(nil)
		}

		sp := &ServicePrincipalChecker{
			log:                      log,
			arocli:                   arocli,
			kubernetescli:            kubernetescli,
			tokenClient:              aad,
			validateServicePrincipal: dynamic,
		}

		t.Run(tt.name, func(t *testing.T) {
			err := sp.Check(ctx)

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%s\n !=\n%s", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateFailedCondition(t *testing.T) {
	for _, tt := range []struct {
		name       string
		messageErr error
		cloudErr   *api.CloudError
		cond       *operatorv1.OperatorCondition
		wantCond   *operatorv1.OperatorCondition
	}{
		{
			name: "pass: successful cloud error condition update",
			cond: &operatorv1.OperatorCondition{
				Type:    arov1alpha1.ServicePrincipalValid,
				Status:  operatorv1.ConditionTrue,
				Message: "service principal is valid",
				Reason:  "CheckDone",
			},
			wantCond: &operatorv1.OperatorCondition{
				Type:    arov1alpha1.ServicePrincipalValid,
				Status:  operatorv1.ConditionFalse,
				Message: "service principal is invalid",
				Reason:  "CheckDone",
			},
			cloudErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    "1",
					Message: "service principal is invalid",
				},
			},
		},
		{
			name: "pass: successful string error condition update",
			cond: &operatorv1.OperatorCondition{
				Type:    arov1alpha1.ServicePrincipalValid,
				Status:  operatorv1.ConditionTrue,
				Message: "service principal is valid",
				Reason:  "CheckDone",
			},
			wantCond: &operatorv1.OperatorCondition{
				Type:    arov1alpha1.ServicePrincipalValid,
				Status:  operatorv1.ConditionFalse,
				Message: "service principal is invalid",
				Reason:  "CheckDone",
			},
			messageErr: fmt.Errorf("service principal is invalid"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cloudErr != nil {
				updateFailedCondition(tt.cond, tt.cloudErr)
			} else if tt.messageErr != nil {
				updateFailedCondition(tt.cond, tt.messageErr)
			}

			if tt.cond.Type != tt.wantCond.Type {
				t.Errorf("\n%s\n !=\n%s", tt.cond.Type, tt.wantCond.Type)
			} else if tt.cond.Message != tt.wantCond.Message {
				t.Errorf("\n%s\n !=\n%s", tt.cond.Message, tt.wantCond.Message)
			}
		})
	}
}
