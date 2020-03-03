package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func TestURLPathsAreLowerCase(t *testing.T) {
	f := &frontend{
		baseLog: logrus.NewEntry(logrus.StandardLogger()),
	}
	router := f.setupRouter()

	varCleanupRe := regexp.MustCompile(`{.*?}`)
	err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			// Ignore the error: it can occur when a route has no path,
			// but there is no way to check it here
			return nil
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
