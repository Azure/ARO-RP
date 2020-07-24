package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *Installer) gatherFailureLogs(ctx context.Context, configClient configclient.Interface) {
	if configClient == nil {
		return
	}

	for _, f := range []func(context.Context, configclient.Interface) (interface{}, error){
		i.logClusterVersion,
		i.logClusterOperators,
	} {
		o, err := f(ctx, configClient)
		if err != nil {
			i.log.Error(err)
			continue
		}
		if o == nil {
			continue
		}

		b, err := json.Marshal(o)
		if err != nil {
			i.log.Error(err)
			continue
		}

		i.log.Printf("%s: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), string(b))
	}
}

func (i *Installer) logClusterVersion(ctx context.Context, configClient configclient.Interface) (interface{}, error) {
	return configClient.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
}

func (i *Installer) logClusterOperators(ctx context.Context, configClient configclient.Interface) (interface{}, error) {
	return configClient.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
}
