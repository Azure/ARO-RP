package frontend

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func (f *frontend) getPreResizeControlPlaneVMsValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	b, err := f._getPreResizeControlPlaneVMsValidation(ctx, resType, resName, resGroupName, resourceID, log)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getPreResizeControlPlaneVMsValidation(ctx context.Context, resType, resName, resGroupName, resourceID string, log *logrus.Entry) ([]byte, error) {
	// TODO: Ensuring Cluster Service Principal is valid

	// TODO: Validating apiserver health

	// WIP: SKU  and Validity
	// -- get availables SKUs in target subscription
	b, err := f._getAdminOpenShiftClusterVMResizeOptions(ctx, resType, resName, resGroupName, resourceID, log)
	if err != nil {
		return nil, err
	}
	// -- TODO: validate available SKU

	return b, nil
}
