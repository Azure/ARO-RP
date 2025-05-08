package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestSystemData(t *testing.T) {
	// extracted from INT:
	// {"createdBy":"f707657a-xxxx-xxxx-xxxx-82704b3a99fd","createdByType":"Application","createdAt":"2021-04-20T11:36:09.3470409Z","lastModifiedBy":"f707657a-bec1-4294-9397-82704b3a99fd","lastModifiedByType":"Application","lastModifiedAt":"2021-04-20T11:36:09.3470409Z"}
	// {"lastModifiedBy":"f707657a-xxxx-xxxx-xxxx-82704b3a99fd","lastModifiedByType":"Application","lastModifiedAt":"2021-04-20T12:14:15.7206198Z"}
	const systemDataCreate = `
{
	"createdBy": "foo@bar.com",
	"createdByType": "Application",
	"createdAt": "2021-01-23T12:34:54.0000000Z",
	"lastModifiedBy": "00000000-0000-0000-0000-000000000000",
	"lastModifiedByType": "Application",
	"lastModifiedAt": "2021-01-23T12:34:54.0000000Z"
}`

	timestamp, err := time.Parse(time.RFC3339, "2021-01-23T12:34:54.0000000Z")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		systemData string
		expect     *api.SystemData
	}{
		{
			name:       "systemData provided",
			systemData: systemDataCreate,
			expect: &api.SystemData{
				CreatedBy:          "foo@bar.com",
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedBy:     "00000000-0000-0000-0000-000000000000",
				LastModifiedAt:     &timestamp,
			},
		},
		{
			name:       "empty",
			systemData: "",
			expect:     &api.SystemData{},
		},
		{
			name:       "invalid",
			systemData: "im_a_potato_not_a_json",
			expect:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodPut, "", bytes.NewReader([]byte("")))
			if err != nil {
				t.Fatal(err)
			}

			r.Header = http.Header{
				"X-Ms-Arm-Resource-System-Data": []string{tt.systemData},
			}

			w := httptest.NewRecorder()
			SystemData(http.HandlerFunc(func(w http.ResponseWriter, _r *http.Request) {
				r = _r
			})).ServeHTTP(w, r)

			ctx := r.Context()

			result, ok := ctx.Value(ContextKeySystemData).(*api.SystemData)
			if ok {
				if !reflect.DeepEqual(result, tt.expect) {
					t.Error(cmp.Diff(result, tt.expect))
				}
			}
		})
	}
}
