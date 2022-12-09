package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

type fakeResponse struct {
	httpResponse *http.Response
	err          error
}

type testClient struct {
	responses []*fakeResponse
}

func (c *testClient) Do(req *http.Request) (*http.Response, error) {
	response := c.responses[0]
	c.responses = c.responses[1:]
	return response.httpResponse, response.err
}

const urltocheck = "https://not-used-in-test.io"

// simulated responses
var (
	okResp = &fakeResponse{
		httpResponse: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(nil),
		},
	}

	badReq = &fakeResponse{
		httpResponse: &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(nil),
		},
	}

	networkUnreach = &fakeResponse{
		err: &url.Error{
			URL: urltocheck,
			Err: &net.OpError{
				Err: os.NewSyscallError("socket", syscall.ENETUNREACH),
			},
		},
	}

	timedoutReq = &fakeResponse{err: context.DeadlineExceeded}
)

func TestInternetCheckerCheck(t *testing.T) {
	ctx := context.Background()

	var testCases = []struct {
		name          string
		responses     []*fakeResponse
		wantErr       string
		wantCondition operatorv1.ConditionStatus
	}{
		{
			name:          "200 OK",
			responses:     []*fakeResponse{okResp},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "bad request",
			responses:     []*fakeResponse{badReq},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "eventual 200 OK",
			responses:     []*fakeResponse{networkUnreach, timedoutReq, okResp},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "eventual bad request",
			responses:     []*fakeResponse{timedoutReq, networkUnreach, badReq},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "timedout request",
			responses:     []*fakeResponse{networkUnreach, timedoutReq, timedoutReq, timedoutReq, timedoutReq, timedoutReq},
			wantErr:       "requeue",
			wantCondition: operatorv1.ConditionFalse,
		},
	}

	roleToConditionTypeMap := map[string]string{
		operator.RoleMaster: arov1alpha1.InternetReachableFromMaster,
		operator.RoleWorker: arov1alpha1.InternetReachableFromWorker,
	}

	for _, testRole := range []string{operator.RoleMaster, operator.RoleWorker} {
		t.Run(testRole, func(t *testing.T) {
			for _, test := range testCases {
				t.Run(test.name, func(t *testing.T) {
					arocli := arofake.NewSimpleClientset(
						&arov1alpha1.Cluster{
							ObjectMeta: metav1.ObjectMeta{
								Name: arov1alpha1.SingletonClusterName,
							},
							Spec: arov1alpha1.ClusterSpec{
								InternetChecker: arov1alpha1.InternetCheckerSpec{
									URLs: []string{urltocheck},
								},
							},
						},
					)

					r := &InternetChecker{
						log:          utillog.GetLogger(),
						role:         testRole,
						checkTimeout: 100 * time.Millisecond,

						arocli:     arocli,
						httpClient: &testClient{responses: test.responses},
					}
					err := r.Check(ctx)
					if err != nil && err.Error() != test.wantErr ||
						err == nil && test.wantErr != "" {
						t.Error(err)
					}

					instance, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
					if err != nil {
						t.Error(err)
					}

					var condition operatorv1.OperatorCondition
					for _, condition = range instance.Status.Conditions {
						if condition.Type == roleToConditionTypeMap[testRole] {
							break
						}
					}

					if condition.Status != test.wantCondition {
						t.Errorf(string(condition.Status))
					}
				})
			}
		})
	}
}
