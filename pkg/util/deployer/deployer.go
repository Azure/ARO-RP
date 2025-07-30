// deployer is used to template and deploy services in an ARO cluster.
// Some example usage can be found in the muo package.
package deployer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

type Deployer interface {
	CreateOrUpdate(context.Context, *arov1alpha1.Cluster, interface{}) error
	Remove(context.Context, interface{}) error
	IsReady(context.Context, string, string) (bool, error)
	Template(interface{}, fs.FS) ([]kruntime.Object, error)
	IsConstraintTemplateReady(context.Context, interface{}) (bool, error)
}

type deployer struct {
	ch        clienthelper.Interface
	fs        fs.FS
	directory string
}

func NewDeployer(log *logrus.Entry, client client.Client, fs fs.FS, directory string) Deployer {
	return &deployer{
		ch:        clienthelper.NewWithClient(log, client),
		fs:        fs,
		directory: directory,
	}
}

func (depl *deployer) Template(data interface{}, fsys fs.FS) ([]kruntime.Object, error) {
	results := make([]kruntime.Object, 0)
	template, err := template.ParseFS(fsys, filepath.Join(depl.directory, "*"))
	if err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	for _, templ := range template.Templates() {
		err := templ.Execute(buffer, data)
		if err != nil {
			return nil, err
		}
		bytes, err := io.ReadAll(buffer)
		if err != nil {
			return nil, err
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(bytes, nil, nil)
		if err != nil {
			o := &unstructured.Unstructured{}
			obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(bytes, nil, o)
			if err != nil {
				return nil, err
			}
			results = append(results, obj)
		} else {
			results = append(results, obj)
		}
		buffer.Reset()
	}

	return results, nil
}

func (depl *deployer) CreateOrUpdate(ctx context.Context, cluster *arov1alpha1.Cluster, config interface{}) error {
	resources, err := depl.Template(config, depl.fs)
	if err != nil {
		return err
	}

	err = dynamichelper.SetControllerReferences(resources, cluster)
	if err != nil {
		return err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	return depl.ch.Ensure(ctx, resources...)
}

func (depl *deployer) Remove(ctx context.Context, data interface{}) error {
	resources, err := depl.Template(data, depl.fs)
	if err != nil {
		return err
	}

	var errs []error
	namespaceName := ""
	for _, obj := range resources {
		nsName, err := depl.removeOne(ctx, obj)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if nsName != "" {
			namespaceName = nsName
		}
	}
	if namespaceName != "" {
		// remove the namespace
		err := depl.ch.EnsureDeleted(ctx, schema.GroupVersionKind{Kind: "Namespace", Version: "v1"}, types.NamespacedName{Name: namespaceName})
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		errContent := []string{"error removing resource:"}
		for _, err := range errs {
			errContent = append(errContent, err.Error())
		}
		return errors.New(strings.Join(errContent, "\n"))
	}

	return nil
}

func (depl *deployer) removeOne(ctx context.Context, obj kruntime.Object) (string, error) {
	o, ok := obj.(client.Object)
	if !ok {
		return "", errors.New("unable to convert into client.Object")
	}
	if o.GetObjectKind().GroupVersionKind().Kind == "Namespace" {
		// don't delete the namespace for now
		return o.GetName(), nil
	}
	errDelete := depl.ch.EnsureDeleted(ctx, o.GetObjectKind().GroupVersionKind(), types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()})
	if errDelete != nil {
		return "", errDelete
	}
	return "", nil
}

func (depl *deployer) IsReady(ctx context.Context, namespace, deploymentName string) (bool, error) {
	d := &appsv1.Deployment{}
	err := depl.ch.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, d)
	switch {
	case kerrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}

	return ready.DeploymentIsReady(d), nil
}

func (depl *deployer) IsConstraintTemplateReady(ctx context.Context, config interface{}) (bool, error) {
	resources, err := depl.Template(config, depl.fs)
	if err != nil {
		return false, err
	}
	for _, resource := range resources {
		if resource.GetObjectKind().GroupVersionKind().Kind == "ConstraintTemplate" {
			r := resource.(*unstructured.Unstructured)

			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(schema.GroupVersionKind{Group: "templates.gatekeeper.sh", Version: "v1", Kind: "ConstraintTemplate"})
			err = depl.ch.Get(ctx, types.NamespacedName{Name: r.GetName()}, u)
			if err != nil {
				return false, err
			}

			ready, ok, err := unstructured.NestedBool(u.Object, "status", "created")
			if !ready || !ok || err != nil {
				return false, err
			}
		}
	}
	return true, nil
}
