package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func SetControllerReferences(resources []kruntime.Object, owner metav1.Object) error {
	for _, resource := range resources {
		r, err := meta.Accessor(resource)
		if err != nil {
			return err
		}

		err = controllerutil.SetControllerReference(owner, r, scheme.Scheme)
		if err != nil {
			return err
		}
	}

	return nil
}

func Prepare(resources []kruntime.Object) error {
	err := hashWorkloadConfigs(resources)
	if err != nil {
		return err
	}

	sort.SliceStable(resources, func(i, j int) bool {
		return createOrder(resources[i], resources[j])
	})

	return nil
}

func addWorkloadHashes(o *metav1.ObjectMeta, t *corev1.PodTemplateSpec, configToHash map[string]string) {
	for _, v := range t.Spec.Volumes {
		if v.Secret != nil {
			if hash, found := configToHash[keyFunc(schema.GroupKind{Kind: "Secret"}, o.Namespace, v.Secret.SecretName)]; found {
				if t.Annotations == nil {
					t.Annotations = map[string]string{}
				}
				t.Annotations["checksum/secret-"+v.Secret.SecretName] = hash
			}
		}

		if v.ConfigMap != nil {
			if hash, found := configToHash[keyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.Namespace, v.ConfigMap.Name)]; found {
				if t.Annotations == nil {
					t.Annotations = map[string]string{}
				}
				t.Annotations["checksum/configmap-"+v.ConfigMap.Name] = hash
			}
		}
	}
}

// hashWorkloadConfigs iterates daemonsets, walks their volumes, and updates
// their pod templates with annotations that include the hashes of the content
// for each configmap or secret.
func hashWorkloadConfigs(resources []kruntime.Object) error {
	// map config resources to their hashed content
	configToHash := map[string]string{}
	for _, o := range resources {
		switch o := o.(type) {
		case *corev1.Secret:
			configToHash[keyFunc(schema.GroupKind{Kind: "Secret"}, o.Namespace, o.Name)] = getHashSecret(o)
		case *corev1.ConfigMap:
			configToHash[keyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.Namespace, o.Name)] = getHashConfigMap(o)
		}
	}

	// iterate over workload controllers and add annotations with the hashes of
	// every config map or secret appropriately to force redeployments on config
	// updates.
	for _, o := range resources {
		switch o := o.(type) {
		case *appsv1.DaemonSet:
			addWorkloadHashes(&o.ObjectMeta, &o.Spec.Template, configToHash)

		case *appsv1.Deployment:
			addWorkloadHashes(&o.ObjectMeta, &o.Spec.Template, configToHash)

		case *appsv1.StatefulSet:
			addWorkloadHashes(&o.ObjectMeta, &o.Spec.Template, configToHash)
		}
	}

	return nil
}

func getHashSecret(o *corev1.Secret) string {
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

func getHashConfigMap(o *corev1.ConfigMap) string {
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

func keyFunc(gk schema.GroupKind, namespace, name string) string {
	s := gk.String()
	if namespace != "" {
		s += "/" + namespace
	}
	s += "/" + name

	return s
}
