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

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	test_log "github.com/Azure/ARO-RP/test/util/log"
)

func TestAdminReply(t *testing.T) {
	for _, tt := range []struct {
		name           string
		header         http.Header
		b              []byte
		err            error
		wantStatusCode int
		wantBody       interface{}
		wantEntries    []test_log.ExpectedLogEntry
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
			wantEntries: []test_log.ExpectedLogEntry{
				{
					Level:   logrus.InfoLevel,
					Message: `404: NotFound: routes.route.openshift.io/doesntexist: routes.route.openshift.io "doesntexist" not found`,
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
			wantEntries: []test_log.ExpectedLogEntry{
				{
					Level:   logrus.InfoLevel,
					Message: `400: RequestNotAllowed: thing: You can't do that.`,
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
			wantEntries: []test_log.ExpectedLogEntry{
				{
					Level:   logrus.ErrorLevel,
					Message: `random error`,
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

			h, log := test_log.NewCapturingLogger()
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

			for _, e := range test_log.AssertLoggingOutput(h, tt.wantEntries) {
				t.Error(e)
			}
		})
	}
}

func TestRoutesAreNamedWithLowerCasePaths(t *testing.T) {
	f := &frontend{
		baseLog: logrus.NewEntry(logrus.StandardLogger()),
	}
	router := f.setupRouter()

	varCleanupRe := regexp.MustCompile(`{.*?}`)
	err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		_, err := route.GetMethods()
		if err != nil {
			if err.Error() == "mux: route doesn't have methods" {
				err = nil
			}
			return err
		}

		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return err
		}

		if route.GetName() == "" {
			t.Errorf("path %s has no name", pathTemplate)
		}

		cleanPathTemplate := varCleanupRe.ReplaceAllString(pathTemplate, "")
		if cleanPathTemplate != strings.ToLower(cleanPathTemplate) {
			t.Error(pathTemplate)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
