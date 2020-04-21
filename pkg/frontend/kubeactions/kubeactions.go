package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type Interface interface {
	Get(ctx context.Context, oc *api.OpenShiftCluster, groupKind, namespace, name string) ([]byte, error)
	List(ctx context.Context, oc *api.OpenShiftCluster, groupKind, namespace string) ([]byte, error)
	CreateOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, obj *unstructured.Unstructured) error
	Delete(ctx context.Context, oc *api.OpenShiftCluster, groupKind, namespace, name string) error
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

func (ka *kubeactions) findGVR(apiresourcelist []*metav1.APIResourceList, groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	var matches []*schema.GroupVersionResource
	for _, apiresources := range apiresourcelist {
		gv, err := schema.ParseGroupVersion(apiresources.GroupVersion)
		if err != nil {
			// this returns a fmt.Errorf which will result in a 500
			// in this case, this seems correct as the GV in kubernetes is wrong
			return nil, err
		}
		if optionalVersion != "" && gv.Version != optionalVersion {
			continue
		}
		for _, apiresource := range apiresources.APIResources {
			if strings.ContainsRune(apiresource.Name, '/') { // no subresources
				continue
			}

			gk := schema.GroupKind{
				Group: gv.Group,
				Kind:  apiresource.Kind,
			}

			if strings.EqualFold(gk.String(), groupKind) {
				return &schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: apiresource.Name,
				}, nil
			}

			if strings.EqualFold(apiresource.Kind, groupKind) {
				matches = append(matches, &schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: apiresource.Name,
				})
			}
		}
	}

	if len(matches) == 0 {
		return nil, api.NewCloudError(
			http.StatusBadRequest, api.CloudErrorCodeInvalidParameter,
			"", "The groupKind '%s' was not found.", groupKind)
	}

	if len(matches) > 1 {
		return nil, api.NewCloudError(
			http.StatusBadRequest, api.CloudErrorCodeInvalidParameter,
			"", "The groupKind '%s' matched multiple groupKinds.", groupKind)
	}

	return matches[0], nil
}

func (ka *kubeactions) getClient(oc *api.OpenShiftCluster) (dynamic.Interface, []*metav1.APIResourceList, error) {
	restconfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return nil, nil, err
	}

	cli, err := discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, nil, err
	}

	_, apiresources, err := cli.ServerGroupsAndResources()
	if err != nil {
		return nil, nil, err
	}

	dyn, err := dynamic.NewForConfig(restconfig)
	if err != nil {
		return nil, nil, err
	}

	return dyn, apiresources, nil
}

func (ka *kubeactions) Get(ctx context.Context, oc *api.OpenShiftCluster, groupKind, namespace, name string) ([]byte, error) {
	dyn, apiresources, err := ka.getClient(oc)
	if err != nil {
		return nil, err
	}

	gvr, err := ka.findGVR(apiresources, groupKind, "")
	if err != nil {
		return nil, err
	}

	un, err := dyn.Resource(*gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}

func (ka *kubeactions) List(ctx context.Context, oc *api.OpenShiftCluster, groupKind, namespace string) ([]byte, error) {
	dyn, apiresources, err := ka.getClient(oc)
	if err != nil {
		return nil, err
	}

	gvr, err := ka.findGVR(apiresources, groupKind, "")
	if err != nil {
		return nil, err
	}

	ul, err := dyn.Resource(*gvr).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ul.MarshalJSON()
}

func (ka *kubeactions) CreateOrUpdate(ctx context.Context, oc *api.OpenShiftCluster, obj *unstructured.Unstructured) error {
	// TODO log changes

	namespace := obj.GetNamespace()
	groupKind := obj.GroupVersionKind().GroupKind().String()

	dyn, apiresources, err := ka.getClient(oc)
	if err != nil {
		return err
	}

	gvr, err := ka.findGVR(apiresources, groupKind, "")
	if err != nil {
		return err
	}

	_, err = dyn.Resource(*gvr).Namespace(namespace).Update(obj, metav1.UpdateOptions{})
	if !errors.IsNotFound(err) {
		return err
	}

	_, err = dyn.Resource(*gvr).Namespace(namespace).Create(obj, metav1.CreateOptions{})
	return err
}

func (ka *kubeactions) Delete(ctx context.Context, oc *api.OpenShiftCluster, groupKind, namespace, name string) error {
	// TODO log changes

	dyn, apiresources, err := ka.getClient(oc)
	if err != nil {
		return err
	}

	gvr, err := ka.findGVR(apiresources, groupKind, "")
	if err != nil {
		return err
	}

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
