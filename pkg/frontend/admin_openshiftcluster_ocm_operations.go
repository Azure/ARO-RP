package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	ocmapi "github.com/Azure/ARO-RP/pkg/util/ocm/api"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/ghodss/yaml"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"path/filepath"
	"strings"
)

func (f *frontend) getOCMClusterInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	ocmActions, err := f.getOCMActions(ctx, r, log)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%v", err)
		return
	}

	clusterInfo, err := ocmActions.GetClusterInfoWithUpgradePolicies(ctx)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%v", err)
		return
	}

	clusterInfoBytes, err := json.Marshal(clusterInfo)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%v", err)
		return
	}

	adminReply(log, w, nil, clusterInfoBytes, nil)
}

func (f *frontend) postAdminOCMCancelUpgradePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	policyID := r.URL.Query().Get("policyID")

	ocmActions, err := f.getOCMActions(ctx, r, log)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%v", err)
		return
	}

	response, err := ocmActions.CancelClusterUpgradePolicy(ctx, policyID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%v", err)
		return
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "%v", err)
		return
	}

	adminReply(log, w, nil, responseBytes, nil)
}

func (f *frontend) getOCMActions(ctx context.Context, r *http.Request, log *logrus.Entry) (adminactions.OCMActions, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	r.URL.Path = filepath.Dir(r.URL.Path)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return nil, err
	}

	clusterID := doc.ID
	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	ps, err := k.KubeGet(ctx, "Secret", "openshift-config", "pull-secret")
	if err != nil {
		return nil, err
	}

	var u unstructured.Unstructured
	if err := json.Unmarshal(ps, &u); err != nil {
		return nil, err
	}
	var secret corev1.Secret
	if err := kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &secret); err != nil {
		return nil, err
	}

	psKeys, err := pullsecret.UnmarshalSecretData(&secret)
	if err != nil {
		return nil, err
	}

	token, ok := psKeys["cloud.openshift.com"]
	if !ok {
		return nil, fmt.Errorf("token not found in pull secret")
	}

	cm, err := k.KubeGet(ctx, "ConfigMap", "openshift-managed-upgrade-operator", "managed-upgrade-operator-config")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cm, &u); err != nil {
		return nil, err
	}
	var configMap corev1.ConfigMap
	if err := kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &configMap); err != nil {
		return nil, err
	}

	configYaml, ok := configMap.Data["config.yaml"]
	if !ok {
		return nil, fmt.Errorf("config.yaml not found")
	}

	var config ocmapi.Config
	err = yaml.Unmarshal([]byte(configYaml), &config)
	if err != nil {
		return nil, err
	}

	// default OCM base URL jic pending upgrade exists
	ocmBaseUrl := "https://api.openshift.com"
	if config.ConfigManager.Source == "OCM" {
		ocmBaseUrl = config.ConfigManager.OcmBaseURL
	}

	return f.ocmActionsFactory(clusterID, ocmBaseUrl, token), nil
}
