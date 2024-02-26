package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"
	"unicode"

	"github.com/Azure/ARO-RP/pkg/api"
)

func CloudError(t *testing.T, err error) {
	cloudErr, ok := err.(*api.CloudError)
	if !ok {
		t.Fatal("must return *api.CloudError")
	}

	if cloudErr.Code == "" {
		t.Error("code is required")
	}
	if cloudErr.Message == "" {
		t.Error("message is required")
	}
	if cloudErr.Message != "" && !unicode.IsUpper(rune(cloudErr.Message[0])) {
		t.Error("message must start with upper case letter")
	}
	if strings.Contains(cloudErr.Message, `"`) {
		t.Error(`message must not contain '"'`)
	}
	if !strings.HasSuffix(cloudErr.Message, ".") {
		t.Error("message must end in '.'")
	}
	if strings.Contains(cloudErr.Target, `"`) {
		t.Error(`target must not contain '"'`)
	}
}
