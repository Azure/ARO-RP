package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *adminactions) GatherFailureLogs(ctx context.Context) {
	for _, f := range []func(context.Context) (interface{}, error){
		a.logClusterVersion,
		a.logClusterOperators,
	} {
		o, err := f(ctx)
		if err != nil {
			a.log.Error(err)
			continue
		}
		if o == nil {
			continue
		}

		b, err := json.Marshal(o)
		if err != nil {
			a.log.Error(err)
			continue
		}

		a.log.Printf("%s: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), string(b))
	}
}

func (a *adminactions) logClusterVersion(ctx context.Context) (interface{}, error) {
	if a.configcli == nil {
		return nil, nil
	}

	return a.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
}

func (a *adminactions) logClusterOperators(ctx context.Context) (interface{}, error) {
	if a.configcli == nil {
		return nil, nil
	}

	return a.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
}
