package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *Installer) gatherFailureLogs(ctx context.Context) {
	for _, f := range []func(context.Context) (interface{}, error){
		i.logClusterVersion,
		i.logClusterOperators,
	} {
		o, err := f(ctx)
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

func (i *Installer) logClusterVersion(ctx context.Context) (interface{}, error) {
	if i.configcli == nil {
		return nil, nil
	}

	return i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
}

func (i *Installer) logClusterOperators(ctx context.Context) (interface{}, error) {
	if i.configcli == nil {
		return nil, nil
	}

	return i.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
}
