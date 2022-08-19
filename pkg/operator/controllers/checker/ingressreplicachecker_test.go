package checker

import (
	"context"
	"testing"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressReplicaChecker(t *testing.T) {
	ctx := context.Background()

	var zeroReplica int32 = 0

	aroCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
	}

	ingressController := &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "openshift-ingress-operator",
		},
		Spec: operatorv1.IngressControllerSpec{
			Replicas: &(zeroReplica),
		},
	}

	arocli := arofake.NewSimpleClientset(aroCluster)

	operatorcli := operatorfake.NewSimpleClientset(ingressController)

	checker := NewIngressReplicaChecker(arocli, operatorcli, "")

	err := checker.Check(ctx)

	if err != nil {
		t.Error(err)
	}

	ingressController, err = operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if *ingressController.Spec.Replicas != 1 {
		t.Errorf("invalid replica count: %d", *ingressController.Spec.Replicas)
	}

}
