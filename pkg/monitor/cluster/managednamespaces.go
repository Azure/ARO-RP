package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

func (mon *Monitor) fetchManagedNamespaces(ctx context.Context) error {
	var cont string

	namespaces := []string{}
	l := &corev1.NamespaceList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return fmt.Errorf("error in list operation: %w", err)
		}

		for _, i := range l.Items {
			if namespace.IsOpenShiftNamespace(i.GetName()) {
				namespaces = append(namespaces, i.GetName())
			}
		}

		cont = l.GetContinue()
		if cont == "" {
			break
		}
	}

	mon.namespacesToMonitor = namespaces
	return nil
}
