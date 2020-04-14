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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
