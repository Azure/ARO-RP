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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const (
	// schedule the check every 5m
	requeueInterval = 5 * time.Minute
)

// if a check fails it is retried using the following parameters
var checkBackoff = wait.Backoff{
	Steps:    5,
	Duration: 5 * time.Second,
	Factor:   2.0,
	Jitter:   0.5,
	Cap:      requeueInterval / 2,
}

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	arocli aroclient.AroV1alpha1Interface
	log    *logrus.Entry
	role   string
}

func NewReconciler(log *logrus.Entry, arocli aroclient.AroV1alpha1Interface, role string) *InternetChecker {
	return &InternetChecker{arocli: arocli, log: log, role: role}
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

	// limit all checks to take no longer than half of the requeueInterval
	ctx, cancel := context.WithTimeout(context.Background(), requeueInterval/2)
	defer cancel()

	checks := make(map[string]chan error)
	for _, url := range instance.Spec.InternetChecker.URLs {
		checks[url] = make(chan error)
		go r.checkWithRetry(ctx, &http.Client{}, url, checkBackoff, checks[url])
	}

	sb := &strings.Builder{}
	checkFailed := false

	for url, ch := range checks {
		if err = <-ch; err != nil {
			fmt.Fprintf(sb, "%s: %s\n", url, err)
			checkFailed = true
		}
	}

	var condition *status.Condition
	if checkFailed {
		condition = &status.Condition{
			Type:    r.conditionType(),
			Status:  corev1.ConditionFalse,
			Message: sb.String(),
			Reason:  "CheckFailed",
		}
	} else {
		condition = &status.Condition{
			Type:    r.conditionType(),
			Status:  corev1.ConditionTrue,
			Message: "Outgoing connection successful.",
			Reason:  "CheckDone",
		}
	}

	err = controllers.SetCondition(r.arocli, condition, r.role)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: requeueInterval, Requeue: true}, nil
}

// check the URL, retrying failed queries a few times
func (r *InternetChecker) checkWithRetry(
	ctx context.Context,
	client simpleHTTPClient,
	url string,
	backoff wait.Backoff,
	ch chan error,
) {
	ch <- retry.OnError(backoff, func(_ error) bool { return true }, func() error {
		localCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(localCtx, http.MethodHead, url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	})
}

// SetupWithManager setup our mananger
func (r *InternetChecker) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Named(controllers.InternetCheckerControllerName).
		Complete(r)
}

func (r *InternetChecker) conditionType() (ctype status.ConditionType) {
	switch r.role {
	case operator.RoleMaster:
		return arov1alpha1.InternetReachableFromMaster
	case operator.RoleWorker:
		return arov1alpha1.InternetReachableFromWorker
	default:
		r.log.Warnf("unknown role %s, assuming worker role", r.role)
		return arov1alpha1.InternetReachableFromWorker
	}
}
