package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	kubernetescli kubernetes.Interface
	arocli        aroclient.AroV1alpha1Interface
	log           *logrus.Entry
	role          string
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.AroV1alpha1Interface, role string) *InternetChecker {
	return &InternetChecker{
		kubernetescli: kubernetescli,
		arocli:        arocli,
		log:           log,
		role:          role,
	}
}

type simpleHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// Reconcile will keep checking that the cluster can connect to essential services.
func (r *InternetChecker) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.Name != arov1alpha1.SingletonClusterName {
		return reconcile.Result{}, nil
	}

	instance, err := r.arocli.Clusters().Get(request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	var condition status.ConditionType
	switch r.role {
	case operator.RoleMaster:
		condition = arov1alpha1.InternetReachableFromMaster
	case operator.RoleWorker:
		condition = arov1alpha1.InternetReachableFromWorker
	}

	urlErrors := map[string]string{}
	for _, url := range instance.Spec.InternetChecker.URLs {
		err = r.check(&http.Client{}, url)
		if err != nil {
			urlErrors[url] = err.Error()
		}
	}

	if len(urlErrors) > 0 {
		sb := &strings.Builder{}
		for url, err := range urlErrors {
			fmt.Fprintf(sb, "%s: %s\n", url, err)
		}
		err = controllers.SetCondition(r.arocli, &status.Condition{
			Type:    condition,
			Status:  corev1.ConditionFalse,
			Message: sb.String(),
			Reason:  "CheckFailed",
		}, r.role)
	} else {
		err = controllers.SetCondition(r.arocli, &status.Condition{
			Type:    condition,
			Status:  corev1.ConditionTrue,
			Message: "Outgoing connection successful.",
			Reason:  "CheckDone",
		}, r.role)
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: time.Minute, Requeue: true}, nil
}

func (r *InternetChecker) check(client simpleHTTPClient, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SetupWithManager setup our mananger
func (r *InternetChecker) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Named(controllers.InternetCheckerControllerName).
		Complete(r)
}
