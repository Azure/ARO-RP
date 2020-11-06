package checker

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

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

// if a check fails it is retried using the following parameters
var checkBackoff = wait.Backoff{
	Steps:    5,
	Duration: 5 * time.Second,
	Factor:   1.5,
	Jitter:   0.5,
	Cap:      1 * time.Minute,
}

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	arocli aroclient.AroV1alpha1Interface
	log    *logrus.Entry
	role   string
}

func NewInternetChecker(log *logrus.Entry, arocli aroclient.AroV1alpha1Interface, role string) *InternetChecker {
	return &InternetChecker{
		arocli: arocli,
		log:    log,
		role:   role,
	}
}

type simpleHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (r *InternetChecker) Name() string {
	return "InternetChecker"
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// Reconcile will keep checking that the cluster can connect to essential services.
func (r *InternetChecker) Check(ctx context.Context) error {
	instance, err := r.arocli.Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ch := make(chan error)
	checkCount := 0
	for _, url := range instance.Spec.InternetChecker.URLs {
		checkCount++
		go func(urlToCheck string) {
			ch <- r.checkWithRetry(&http.Client{}, urlToCheck, checkBackoff)
		}(url)
	}

	sb := &strings.Builder{}
	checkFailed := false

	for i := 0; i < checkCount; i++ {
		if err = <-ch; err != nil {
			r.log.Infof("URL check failed with error %s", err)
			fmt.Fprintf(sb, "%s\n", err)
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
			Message: "Outgoing connection successful",
			Reason:  "CheckDone",
		}

	}

	return controllers.SetCondition(ctx, r.arocli, condition, r.role)
}

// check the URL, retrying a failed query a few times according to the given backoff
func (r *InternetChecker) checkWithRetry(client simpleHTTPClient, url string, backoff wait.Backoff) error {
	return retry.OnError(backoff, func(_ error) bool { return true }, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return fmt.Errorf("%s: %s", url, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %s", url, err)
		}
		defer resp.Body.Close()

		return nil
	})
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
