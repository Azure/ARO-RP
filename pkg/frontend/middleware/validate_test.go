package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/jim-minter/rp/pkg/api"
)

var testPath = "/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}"

// Handler is responsible for defining a HTTP corresponding handler.
type Handler struct {
	Func http.HandlerFunc
}

// AddRoute adds the handler's route the to the router.
func (h Handler) AddRoute(r *mux.Router, path, method string) {
	r.NewRoute().Path(path).Methods(method).
		HandlerFunc(h.Func)
}

func GetTestHandler() Handler {
	return Handler{
		Func: func(rw http.ResponseWriter, req *http.Request) {
			rw.Write([]byte("ack"))
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantStatusCode int
		expectMessage  string
	}{
		{
			name:           "Invalid subscription",
			url:            "/subscriptions/subscriptionId/resourcegroups/resourceGroupName/providers/resourceProviderNamespace/resourceType/resourceName?api-version=test",
			wantStatusCode: http.StatusNotFound,
			expectMessage:  api.CloudErrorCodeInvalidSubscriptionID,
		},
		{
			name:           "Invalid resourceGroupName",
			url:            "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/resourceGroupName/providers/resourceProviderNamespace/resourceType/resourceName?api-version=test",
			wantStatusCode: http.StatusNotFound,
			expectMessage:  api.CloudErrorCodeResourceGroupNotFound,
		},
		{
			name:           "Invalid resourceProviderNamespace",
			url:            "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/test-resourcegroup/providers/resourceProviderNamespace/resourceType/resourceName?api-version=test",
			wantStatusCode: http.StatusNotFound,
			expectMessage:  api.CloudErrorCodeInvalidResourceNamespace,
		},
		{
			name:           "Invalid resourceType",
			url:            "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/test-resourcegroup/providers/microsoft.redhatopenshift/resourceType/resourceName?api-version=test",
			wantStatusCode: http.StatusNotFound,
			expectMessage:  api.CloudErrorCodeInvalidResourceType,
		},
		{
			name:           "Invalid resourceName",
			url:            "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/test-resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/myMYCLUSTER?api-version=test",
			wantStatusCode: http.StatusNotFound,
			expectMessage:  api.CloudErrorCodeResourceNotFound,
		},
		{
			name:           "valid case",
			url:            "/subscriptions/42d9eac4-d29a-4d6e-9e26-3439758b1491/resourcegroups/test-resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/mycluster?api-version=2019-12-31-preview",
			wantStatusCode: http.StatusOK,
			expectMessage:  "ack",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := mux.NewRouter()

			GetTestHandler().AddRoute(r, testPath, http.MethodPost)
			r.Use(Validate)

			req := httptest.NewRequest(http.MethodPost, test.url, bytes.NewBuffer([]byte("")))

			r.ServeHTTP(w, req)

			if test.wantStatusCode != w.Code {
				t.Errorf("test %s failed %d != %d", test.name, test.wantStatusCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), test.expectMessage) {
				t.Errorf("test %s failed %s does not contain %s", test.name, w.Body.String(), test.expectMessage)
			}
		})
	}
}
