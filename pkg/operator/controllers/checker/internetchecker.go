package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	log *logrus.Entry

	arocli aroclient.Interface

	role string
}

func NewInternetChecker(log *logrus.Entry, arocli aroclient.Interface, role string) *InternetChecker {
	return &InternetChecker{
		log:    log,
		arocli: arocli,
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
	cli := &http.Client{
		Transport: &http.Transport{
			// We set DisableKeepAlives for two reasons:
			//
			// 1. If we're talking HTTP/2 and the remote end blackholes traffic,
			// Go has a bug whereby it doesn't reset the connection after a
			// timeout (https://github.com/golang/go/issues/36026).  If this
			// happens, we never have a chance to get healthy.  We have
			// specifically seen this with gcs.prod.monitoring.core.windows.net
			// in Korea Central, which currently has a bad server which when we
			// hit it causes our cluster creations to fail.
			//
			// 2. We *want* to evaluate our capability to successfully create
			// *new* connections to internet endpoints anyway.
			DisableKeepAlives: true,
		},
	}

	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ch := make(chan error)
	checkCount := 0
	for _, url := range instance.Spec.InternetChecker.URLs {
		checkCount++
		go func(urlToCheck string) {
			ch <- r.checkWithRetry(cli, urlToCheck, time.Minute)
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

	var condition *operatorv1.OperatorCondition

	if checkFailed {
		condition = &operatorv1.OperatorCondition{
			Type:    r.conditionType(),
			Status:  operatorv1.ConditionFalse,
			Message: sb.String(),
			Reason:  "CheckFailed",
		}
	} else {
		condition = &operatorv1.OperatorCondition{
			Type:    r.conditionType(),
			Status:  operatorv1.ConditionTrue,
			Message: "Outgoing connection successful",
			Reason:  "CheckDone",
		}

	}

	err = conditions.SetCondition(ctx, r.arocli, condition, r.role)
	if err != nil {
		return err
	}

	if checkFailed {
		return errRequeue
	}

	return nil
}

// check the URL, retrying a failed query a few times
func (r *InternetChecker) checkWithRetry(client simpleHTTPClient, url string, timeout time.Duration) error {
	var err error

	for i := 0; i < 6; i++ {
		err = r.checkOnce(client, url, timeout/6)
		if err == nil {
			return nil
		}
	}

	return err
}

// checkOnce checks a given url.  The check both times out after a given timeout
// *and* will wait for the timeout if it fails, so that we don't hit endpoints
// too much.
func (r *InternetChecker) checkOnce(client simpleHTTPClient, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		<-ctx.Done()
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		<-ctx.Done()
		return fmt.Errorf("%s: %s", url, err)
	}

	resp.Body.Close()
	return nil
}

func (r *InternetChecker) conditionType() (ctype string) {
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
