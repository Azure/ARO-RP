package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/util/jsonpath"
)

// HashWorkloadConfigs iterates over all workload controllers (deployments,
// daemonsets, statefulsets), walks their volumes, and updates their pod
// templates with annotations that include the hashes of the content for
// each configmap or secret.
func HashWorkloadConfigs(resources []*unstructured.Unstructured) {
	// map config resources to their hashed content
	configToHash := make(map[string]string)
	for _, o := range resources {
		gk := o.GroupVersionKind().GroupKind()

		if gk.String() != "Secret" &&
			gk.String() != "ConfigMap" {
			continue
		}

		configToHash[keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] = getHash(o)
	}

	// iterate over all workload controllers and add annotations with the hashes
	// of every config map or secret appropriately to force redeployments on config
	// updates.
	for _, o := range resources {
		gk := o.GroupVersionKind().GroupKind()

		if gk.String() != "DaemonSet.apps" &&
			gk.String() != "Deployment.apps" &&
			gk.String() != "StatefulSet.apps" {
			continue
		}

		volumes := jsonpath.MustCompile("$.spec.template.spec.volumes.*").Get(o.Object)
		for _, v := range volumes {
			v := v.(map[string]interface{})

			if secretData, found := v["secret"]; found {
				secretName := jsonpath.MustCompile("$.secretName").MustGetString(secretData)
				key := fmt.Sprintf("checksum/secret-%s", secretName)
				secretKey := keyFunc(schema.GroupKind{Kind: "Secret"}, o.GetNamespace(), secretName)
				if hash, found := configToHash[secretKey]; found {
					setPodTemplateAnnotation(key, hash, o)
				}
			}

			if configMapData, found := v["configMap"]; found {
				configMapName := jsonpath.MustCompile("$.name").MustGetString(configMapData)
				key := fmt.Sprintf("checksum/configmap-%s", configMapName)
				configMapKey := keyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.GetNamespace(), configMapName)
				if hash, found := configToHash[configMapKey]; found {
					setPodTemplateAnnotation(key, hash, o)
				}
			}
		}
	}
}

func getHash(o *unstructured.Unstructured) string {
	var content map[string]interface{}
	for _, v := range jsonpath.MustCompile("$.data").Get(o.Object) {
		content = v.(map[string]interface{})
	}
	for _, v := range jsonpath.MustCompile("$.stringData").Get(o.Object) {
		content = v.(map[string]interface{})
	}
	// sort config content appropriately
	var keys []string
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(h, "%s: %#v", key, content[key])
	}

	return hex.EncodeToString(h.Sum(nil))
}

// setPodTemplateAnnotation sets the provided key-value pair as an annotation
// inside the provided object's pod template.
func setPodTemplateAnnotation(key, value string, o *unstructured.Unstructured) {
	annotations, _, _ := unstructured.NestedStringMap(o.Object, "spec", "template", "metadata", "annotations")
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	unstructured.SetNestedStringMap(o.Object, annotations, "spec", "template", "metadata", "annotations")
}
