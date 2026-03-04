package frontend

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

const envLocalDevMockHive = "LOCALDEV_MOCK_HIVE"

func (f *frontend) adminHiveK8sObjectsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resource := chi.URLParam(r, "resource")
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")

	if namespace == "" {
		adminReply(log, w, nil, nil,
			api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidRequestContent,
				"",
				"namespace is required",
			))
		return
	}

	// Local dev mock response
	if os.Getenv(envLocalDevMockHive) == "true" {
		log.Warn("using LOCALDEV_MOCK_HIVE mock response")

		resp := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"metadata": map[string]string{
						"name":      "local-dev-pod",
						"namespace": namespace,
					},
				},
			},
		}

		b, err := json.Marshal(resp)
		if err != nil {
			adminReply(log, w, nil, nil, err)
			return
		}

		adminReply(log, w, nil, b, nil)
		return
	}

	if f.hiveK8sObjectManager == nil {
		adminReply(log, w, nil, nil,
			api.NewCloudError(
				http.StatusNotImplemented,
				api.CloudErrorCodeInternalServerError,
				"",
				"hive k8s object manager not configured",
			))
		return
	}

	var (
		b   []byte
		err error
	)

	if name != "" {
		b, err = f.hiveK8sObjectManager.Get(ctx, resource, namespace, name)
	} else {
		b, err = f.hiveK8sObjectManager.List(ctx, resource, namespace)
	}

	adminReply(log, w, nil, b, err)
}
