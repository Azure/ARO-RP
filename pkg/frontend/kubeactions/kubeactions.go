package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-cmp/cmp"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type Interface interface {
	Get(ctx context.Context, oc *api.OpenShiftCluster, kind, namespace, name string) ([]byte, error)
	List(ctx context.Context, oc *api.OpenShiftCluster, kind, namespace string) ([]byte, error)
	CreateOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, body []byte) error
	Delete(ctx context.Context, oc *api.OpenShiftCluster, kind, namespace, name string) error
	ClusterUpgrade(ctx context.Context, oc *api.OpenShiftCluster) error
	MustGather(ctx context.Context, oc *api.OpenShiftCluster, w io.Writer) error
}

type kubeactions struct {
	log *logrus.Entry
	env env.Interface
}

func New(log *logrus.Entry, env env.Interface) Interface {
	return &kubeactions{
		log: log,
		env: env,
	}
}

func (ka *kubeactions) findGVR(grs []*restmapper.APIGroupResources, kind string) []*schema.GroupVersionResource {
	var matches []*schema.GroupVersionResource

	for _, gr := range grs {
		for version, resources := range gr.VersionedResources {
			if version != gr.Group.PreferredVersion.Version {
				continue
			}

			for _, resource := range resources {
				if strings.ContainsRune(resource.Name, '/') { // no subresources
					continue
				}

				gk := schema.GroupKind{
					Group: gr.Group.Name,
					Kind:  resource.Kind,
				}

				if strings.EqualFold(gk.String(), kind) {
					return []*schema.GroupVersionResource{
						{
							Group:    gr.Group.Name,
							Version:  version,
							Resource: resource.Name,
						},
					}
				}

				if strings.EqualFold(resource.Kind, kind) {
					matches = append(matches, &schema.GroupVersionResource{
						Group:    gr.Group.Name,
						Version:  version,
						Resource: resource.Name,
					})
				}
			}
		}
	}

	return matches
}

func (ka *kubeactions) getClient(oc *api.OpenShiftCluster) (dynamic.Interface, []*restmapper.APIGroupResources, error) {
	restconfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return nil, nil, err
	}

	cli, err := discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, nil, err
	}

	grs, err := restmapper.GetAPIGroupResources(cli)
	if err != nil {
		return nil, nil, err
	}

	dyn, err := dynamic.NewForConfig(restconfig)
	if err != nil {
		return nil, nil, err
	}

	return dyn, grs, nil
}

func (ka *kubeactions) Get(ctx context.Context, oc *api.OpenShiftCluster, kind, namespace, name string) ([]byte, error) {
	dyn, grs, err := ka.getClient(oc)
	if err != nil {
		return nil, err
	}

	gvrs := ka.findGVR(grs, kind)

	if len(gvrs) == 0 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' was not found.", kind)
	}

	if len(gvrs) > 1 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' matched multiple GroupKinds.", kind)
	}

	gvr := gvrs[0]

	un, err := dyn.Resource(*gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}

func (ka *kubeactions) List(ctx context.Context, oc *api.OpenShiftCluster, kind, namespace string) ([]byte, error) {
	dyn, grs, err := ka.getClient(oc)
	if err != nil {
		return nil, err
	}

	gvrs := ka.findGVR(grs, kind)
	if len(gvrs) == 0 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' was not found.", kind)
	}

	if len(gvrs) > 1 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' matched multiple GroupKinds.", kind)
	}

	gvr := gvrs[0]

	ul, err := dyn.Resource(*gvr).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ul.MarshalJSON()
}

func printDiff(log *logrus.Entry, existing, o *unstructured.Unstructured) {
	if diff := cmp.Diff(*existing, *o); diff != "" {
		log.Info(diff)
	}
}

func (ka *kubeactions) createOrUpdateOne(ctx context.Context, dyn dynamic.Interface, grs []*restmapper.APIGroupResources, un *unstructured.Unstructured) error {
	if strings.EqualFold(un.GetKind(), "secret") {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to secrets is forbidden.")
	}

	gvrs := ka.findGVR(grs, un.GetKind())
	if len(gvrs) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' was not found.", un.GetKind())
	}
	if len(gvrs) > 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' matched multiple GroupKinds.", un.GetKind())
	}
	gvr := gvrs[0]

	_, err := dyn.Resource(*gvr).Namespace(un.GetNamespace()).Get(un.GetName(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = dyn.Resource(*gvr).Namespace(un.GetNamespace()).Create(un, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		// TODO how to log the changes made : printDiff(ka.log, before, after)
		return nil
	}

	_, err = dyn.Resource(*gvr).Namespace(un.GetNamespace()).Update(un, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	// TODO how to log the changes made : printDiff(ka.log, before, after)
	return nil
}

func (ka *kubeactions) CreateOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, body []byte) error {
	dyn, grs, err := ka.getClient(oc)
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(body)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}
	return ka.createOrUpdateOne(ctx, dyn, grs, obj)
}

func (ka *kubeactions) Delete(ctx context.Context, oc *api.OpenShiftCluster, kind, namespace, name string) error {
	dyn, grs, err := ka.getClient(oc)
	if err != nil {
		return err
	}

	gvrs := ka.findGVR(grs, kind)

	if len(gvrs) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' was not found.", kind)
	}

	if len(gvrs) > 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The kind '%s' matched multiple GroupKinds.", kind)
	}

	gvr := gvrs[0]

	// TODO log the deletion
	return dyn.Resource(*gvr).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
}

// ClusterUpgrade posts the new version and image to the cluster-version-operator
// which will effect the upgrade.
func (ka *kubeactions) ClusterUpgrade(ctx context.Context, oc *api.OpenShiftCluster) error {
	restconfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return err
	}

	configcli, err := configclient.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: version.OpenShiftVersion,
			Image:   version.OpenShiftPullSpec,
		}

		_, err = configcli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
