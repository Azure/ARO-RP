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
	"reflect"
	"strings"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
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
	client    client.Client
	dh        dynamichelper.Interface
	fs        fs.FS
	directory string
}

func NewDeployer(client client.Client, dh dynamichelper.Interface, fs fs.FS, directory string) Deployer {
	return &deployer{
		client:    client,
		dh:        dh,
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
			return nil, err
		}
		results = append(results, obj)
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

	return depl.dh.Ensure(ctx, resources...)
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
		err := depl.dh.EnsureDeleted(ctx, "Namespace", "", namespaceName)
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
	// remove everything we created that has name and ns
	nameValue, err := getField(obj, "Name")
	if err != nil {
		return "", err
	}
	nsValue, err := getField(obj, "Namespace")
	if err != nil {
		return "", err
	}
	name := nameValue.String()
	ns := nsValue.String()
	if obj.GetObjectKind().GroupVersionKind().GroupKind().String() == "Namespace" {
		// don't delete the namespace for now
		return name, nil
	}
	errDelete := depl.dh.EnsureDeletedGVR(ctx, obj.GetObjectKind().GroupVersionKind().GroupKind().String(), ns, name, "")
	if errDelete != nil {
		return "", errDelete
	}
	return "", nil
}

func getField(obj interface{}, fieldName string) (reflect.Value, error) {
	if fieldName == "" {
		return reflect.Value{}, errors.New("empty field name")
	}
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return reflect.Value{}, errors.New("obj not ptr")
	}
	elem := reflect.ValueOf(obj).Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("obj not pointing to struct")
	}
	field := elem.FieldByName(fieldName)
	if !field.IsValid() {
		return reflect.Value{}, errors.New("not found field: " + fieldName)
	}
	return field, nil
}

func (depl *deployer) IsReady(ctx context.Context, namespace, deploymentName string) (bool, error) {
	d := &appsv1.Deployment{}
	err := depl.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, d)
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
		if reflect.TypeOf(resource).String() == "*v1.ConstraintTemplate" {
			name, err := getField(resource, "Name")
			if err != nil {
				return false, err
			}
			ready, err := depl.dh.IsConstraintTemplateReady(ctx, name.String())
			if !ready || err != nil {
				return ready, err
			}
		}
	}
	return true, nil
}
