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

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/ready"

	"github.com/sirupsen/logrus"
)

type Deployer interface {
	CreateOrUpdate(context.Context, *arov1alpha1.Cluster, interface{}) error
	Remove(context.Context, interface{}) error
	IsReady(context.Context, string, string) (bool, error)
	Template(interface{}, fs.FS) ([]kruntime.Object, error)
}

type deployer struct {
	kubernetescli kubernetes.Interface
	dh            dynamichelper.Interface
	fs            fs.FS
	directory     string
	unstructured  bool
}

func NewDeployer(kubernetescli kubernetes.Interface, dh dynamichelper.Interface, fs fs.FS, directory string) Deployer {
	return &deployer{
		kubernetescli: kubernetescli,
		dh:            dh,
		fs:            fs,
		directory:     directory,
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
			// for unrecognised ones try unstructured way
			logrus.Printf("\x1b[%dm scheme.Codecs decode failed try unstructured way for %v err %v\x1b[0m", 31, templ.Name, err)
			uns := dynamichelper.UnstructuredObj{}
			err := uns.DecodeUnstructured(bytes)
			if err != nil {
				return nil, err
			}
			depl.unstructured = true
			results = append(results, uns)
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
		logrus.Printf("\x1b[%dm Template failed %v\x1b[0m", 31, err)
		return err
	}

	// this call fails with "object does not implement the Object interfaces" for ConstraintTemplate if not adding templatesv1beta1 in scheme.go
	err = dynamichelper.SetControllerReferences(resources, cluster)
	if err != nil {
		logrus.Printf("\x1b[%dm SetControllerReferences failed %v\x1b[0m", 31, err)
		return err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		logrus.Printf("\x1b[%dm Prepare failed %v\x1b[0m", 31, err)
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
		// remove everything we created that has name and ns
		if getName, getNs := reflect.ValueOf(obj).MethodByName("GetName"), reflect.ValueOf(obj).MethodByName("GetNamespace"); getName != reflect.ValueOf(nil) && getNs != reflect.ValueOf(nil) {
			name := getName.Call(nil)
			ns := getNs.Call(nil)
			if reflect.TypeOf(obj).String() != "*v1.Namespace" { // dont remove the ns for now
				err := depl.dh.EnsureDeletedGVR(ctx, obj.GetObjectKind().GroupVersionKind().GroupKind().String(), ns[0].String(), name[0].String(), "")
				if err != nil {
					errs = append(errs, err)
				}
			} else {
				namespaceName = name[0].String()
			}
		}
	}
	// remove ns finally
	if namespaceName != "" {
		err := depl.dh.EnsureDeleted(ctx, "Namespace", "", namespaceName)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		errContent := []string{"error removing deployment/unstructured:"}
		for _, err := range errs {
			errContent = append(errContent, err.Error())
		}
		return errors.New(strings.Join(errContent, "\n"))
	}

	return nil
}

func (depl *deployer) IsReady(ctx context.Context, namespace, deploymentName string) (bool, error) {
	if !depl.unstructured {
		return ready.CheckDeploymentIsReady(ctx, depl.kubernetescli.AppsV1().Deployments(namespace), deploymentName)()
	}
	return true, nil
}
