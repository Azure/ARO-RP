package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestAdminReply(t *testing.T) {
	for _, tt := range []struct {
		name           string
		header         http.Header
		b              []byte
		err            error
		wantStatusCode int
		wantBody       interface{}
		wantEntries    []map[string]types.GomegaMatcher
	}{
		{
			name: "kubernetes error",
			err: &kerrors.StatusError{
				ErrStatus: metav1.Status{
					Message: `routes.route.openshift.io "doesntexist" not found`,
					Reason:  metav1.StatusReasonNotFound,
					Details: &metav1.StatusDetails{
						Name:  "doesntexist",
						Group: "route.openshift.io",
						Kind:  "routes",
					},
					Code: http.StatusNotFound,
				},
			},
			wantStatusCode: http.StatusNotFound,
			wantBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "NotFound",
					"message": `routes.route.openshift.io "doesntexist" not found`,
					"target":  "routes.route.openshift.io/doesntexist",
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`404: NotFound: routes.route.openshift.io/doesntexist: routes.route.openshift.io "doesntexist" not found`),
				},
			},
		},
		{
			name: "cloud error",
			err: &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeRequestNotAllowed,
					Message: "You can't do that.",
					Target:  "thing",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    api.CloudErrorCodeRequestNotAllowed,
					"message": "You can't do that.",
					"target":  "thing",
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal(`400: RequestNotAllowed: thing: You can't do that.`),
				},
			},
		},
		{
			name:           "status code error",
			err:            statusCodeError(http.StatusTeapot),
			wantStatusCode: http.StatusTeapot,
		},
		{
			name:           "other error",
			err:            errors.New("random error"),
			wantStatusCode: http.StatusInternalServerError,
			wantBody: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    api.CloudErrorCodeInternalServerError,
					"message": "Internal server error.",
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal(`random error`),
				},
			},
		},
		{
			name: "normal output",
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			b:              []byte("{}"),
			wantStatusCode: http.StatusOK,
			wantBody:       map[string]interface{}{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.header == nil {
				tt.header = http.Header{}
			}

			h, log := testlog.New()
			w := &httptest.ResponseRecorder{
				Body: &bytes.Buffer{},
			}

			adminReply(log, w, tt.header, tt.b, tt.err)

			resp := w.Result()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			if !reflect.DeepEqual(resp.Header, tt.header) {
				t.Error(resp.Header)
			}

			if tt.wantBody != nil {
				var body interface{}
				err := json.Unmarshal(w.Body.Bytes(), &body)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(body, tt.wantBody) {
					t.Error(w.Body.String())
				}
			} else {
				if w.Body.Len() > 0 {
					t.Error(w.Body.String())
				}
			}

			err := testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestFrontendOperationResultLog(t *testing.T) {
	for _, tt := range []struct {
		name     string
		err      error
		wantData logrus.Fields
	}{
		{
			name: "Success Status Code",
			err: &api.CloudError{
				StatusCode: http.StatusNoContent,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeResourceNotFound,
					Message: "This is not a real error",
					Target:  "target",
				},
			},
			wantData: logrus.Fields{
				"LOGKIND":       "frontendqos",
				"operationType": "method",
				"resultType":    utillog.SuccessResultType,
			},
		},
		{
			name: "User Error Status Code",
			err: &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeResourceNotFound,
					Message: "This is an user error result type",
					Target:  "target",
				},
			},
			wantData: logrus.Fields{
				"LOGKIND":       "frontendqos",
				"operationType": "method",
				"resultType":    utillog.UserErrorResultType,
			},
		},
		{
			name: "Server Error Status Code",
			err: &api.CloudError{
				StatusCode: http.StatusInternalServerError,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "This is a server error result type",
					Target:  "target",
				},
			},
			wantData: logrus.Fields{
				"LOGKIND":       "frontendqos",
				"operationType": "method",
				"resultType":    utillog.ServerErrorResultType,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := testlog.New()

			frontendOperationResultLog(log, "method", tt.err)

			entry := h.LastEntry()
			if entry == nil {
				t.Fatal("Expected log entry, got nil")
			}

			for key, value := range tt.wantData {
				if entry.Data[key] != value {
					t.Errorf("Unexpected value for key %s, got %v, want %v", key, entry.Data[key], value)
				}
			}
		})
	}
}

func TestRoutesAreNamedWithLowerCasePaths(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

	f := &frontend{
		baseLog: logrus.NewEntry(logrus.StandardLogger()),
		env:     _env,
	}
	router := f.setupRouter()

	routes := router.Routes()
	for _, route := range routes {
		if !routeHasHandlers(route) {
			t.Errorf("no handler for some routes in %s", route)
		}
		if !routeIsAllLowercase(route) {
			t.Errorf("route %s is not all lowercase", route.Pattern)
		}
	}
}

func routeHasHandlers(route chi.Route) bool {
	if route.SubRoutes == nil {
		return true
	}

	if len(route.SubRoutes.Routes()) == 0 {
		return len(route.Handlers) > 0
	}

	for _, v := range route.SubRoutes.Routes() {
		if !routeHasHandlers(v) {
			return false
		}
	}

	return true
}

func routeIsAllLowercase(route chi.Route) bool {
	varCleanupRe := regexp.MustCompile(`{.*?}`)
	pattern := varCleanupRe.ReplaceAllString(route.Pattern, "")
	if route.SubRoutes == nil {
		return true
	}
	if len(route.SubRoutes.Routes()) == 0 {
		return pattern == strings.ToLower(pattern)
	}

	for _, v := range route.SubRoutes.Routes() {
		if !(pattern == strings.ToLower(pattern)) || !routeIsAllLowercase(v) {
			return false
		}
	}

	return true
}
