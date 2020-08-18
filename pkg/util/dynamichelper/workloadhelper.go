package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HashWorkloadConfigs iterates daemonsets, walks their volumes, and updates
// their pod templates with annotations that include the hashes of the content
// for each configmap or secret.
func HashWorkloadConfigs(resources []runtime.Object) error {
	// map config resources to their hashed content
	configToHash := map[string]string{}
	for _, o := range resources {
		switch o := o.(type) {
		case *v1.Secret:
			configToHash[keyFunc(schema.GroupKind{Kind: "Secret"}, o.Namespace, o.Name)] = getHashSecret(o)
		case *v1.ConfigMap:
			configToHash[keyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.Namespace, o.Name)] = getHashConfigMap(o)
		}
	}

	// iterate over workload controllers and add annotations with the hashes of
	// every config map or secret appropriately to force redeployments on config
	// updates.
	for _, o := range resources {
		switch o := o.(type) {
		case *appsv1.DaemonSet:
			for _, v := range o.Spec.Template.Spec.Volumes {
				if v.Secret != nil {
					if hash, found := configToHash[keyFunc(schema.GroupKind{Kind: "Secret"}, o.Namespace, v.Secret.SecretName)]; found {
						if o.Spec.Template.Annotations == nil {
							o.Spec.Template.Annotations = map[string]string{}
						}
						o.Spec.Template.Annotations["checksum/secret-"+v.Secret.SecretName] = hash
					}
				}

				if v.ConfigMap != nil {
					if hash, found := configToHash[keyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.Namespace, v.ConfigMap.Name)]; found {
						if o.Spec.Template.Annotations == nil {
							o.Spec.Template.Annotations = map[string]string{}
						}
						o.Spec.Template.Annotations["checksum/configmap-"+v.ConfigMap.Name] = hash
					}
				}
			}

		case *appsv1.Deployment, *appsv1.StatefulSet:
			// TODO: add as/when needed
			return fmt.Errorf("unimplemented: %T", o)
		}
	}

	return nil
}

func getHashSecret(o *v1.Secret) string {
	keys := make([]string, 0, len(o.Data))
	for key := range o.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(h, "%s: %s\n", key, string(o.Data[key]))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func getHashConfigMap(o *v1.ConfigMap) string {
	keys := make([]string, 0, len(o.Data))
	for key := range o.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(h, "%s: %s\n", key, o.Data[key])
	}

	return hex.EncodeToString(h.Sum(nil))
}
