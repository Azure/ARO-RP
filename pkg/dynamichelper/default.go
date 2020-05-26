package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/util/jsonpath"
)

func defaultContainerSpec(obj map[string]interface{}) {
	jsonpath.MustCompile("$.livenessProbe.failureThreshold").DeleteIfMatch(obj, int64(3))
	jsonpath.MustCompile("$.livenessProbe.periodSeconds").DeleteIfMatch(obj, int64(10))
	jsonpath.MustCompile("$.livenessProbe.successThreshold").DeleteIfMatch(obj, int64(1))
	jsonpath.MustCompile("$.livenessProbe.timeoutSeconds").DeleteIfMatch(obj, int64(1))
	jsonpath.MustCompile("$.ports.*.protocol").DeleteIfMatch(obj, "TCP")
	jsonpath.MustCompile("$.readinessProbe.failureThreshold").DeleteIfMatch(obj, int64(3))
	jsonpath.MustCompile("$.readinessProbe.periodSeconds").DeleteIfMatch(obj, int64(10))
	jsonpath.MustCompile("$.readinessProbe.successThreshold").DeleteIfMatch(obj, int64(1))
	jsonpath.MustCompile("$.readinessProbe.timeoutSeconds").DeleteIfMatch(obj, int64(1))
	jsonpath.MustCompile("$.terminationMessagePath").DeleteIfMatch(obj, "/dev/termination-log")
	jsonpath.MustCompile("$.terminationMessagePolicy").DeleteIfMatch(obj, "File")
}

func defaultPodSpec(obj map[string]interface{}) {
	for _, c := range jsonpath.MustCompile("$.containers.*").Get(obj) {
		defaultContainerSpec(c.(map[string]interface{}))
	}
	for _, c := range jsonpath.MustCompile("$.initContainers.*").Get(obj) {
		defaultContainerSpec(c.(map[string]interface{}))
	}
	jsonpath.MustCompile("$.dnsPolicy").DeleteIfMatch(obj, "ClusterFirst")
	jsonpath.MustCompile("$.restartPolicy").DeleteIfMatch(obj, "Always")
	jsonpath.MustCompile("$.schedulerName").DeleteIfMatch(obj, "default-scheduler")
	jsonpath.MustCompile("$.securityContext").DeleteIfMatch(obj, map[string]interface{}{})
	jsonpath.MustCompile("$.serviceAccount").Delete(obj) // deprecated alias of serviceAccountName
	jsonpath.MustCompile("$.terminationGracePeriodSeconds").DeleteIfMatch(obj, int64(30))
	jsonpath.MustCompile("$.volumes.*.configMap.defaultMode").DeleteIfMatch(obj, int64(0644))
	jsonpath.MustCompile("$.volumes.*.hostPath.type").DeleteIfMatch(obj, "")
	jsonpath.MustCompile("$.volumes.*.secret.defaultMode").DeleteIfMatch(obj, int64(0644))
}

// defaults removes default values, which don't have to be specified in sync pods config
// and are filled when applying the configuration to a cluster
func defaults(o unstructured.Unstructured) {
	gk := o.GroupVersionKind().GroupKind()

	switch gk.String() {
	case "CronJob.batch":
		for _, c := range jsonpath.MustCompile("$.spec.jobTemplate.spec.template.spec").Get(o.Object) {
			defaultPodSpec(c.(map[string]interface{}))
		}

		jsonpath.MustCompile("$.spec.concurrencyPolicy").DeleteIfMatch(o.Object, "Allow")
		jsonpath.MustCompile("$.spec.failedJobsHistoryLimit").DeleteIfMatch(o.Object, int64(1))
		jsonpath.MustCompile("$.spec.successfulJobsHistoryLimit").DeleteIfMatch(o.Object, int64(3))
		jsonpath.MustCompile("$.spec.suspend").DeleteIfMatch(o.Object, false)

	case "CustomResourceDefinition.apiextensions.k8s.io":
		jsonpath.MustCompile("$.spec.conversion").DeleteIfMatch(o.Object, map[string]interface{}{"strategy": string("None")})
		jsonpath.MustCompile("$.spec.preserveUnknownFields").DeleteIfMatch(o.Object, true)

	case "DaemonSet.apps":
		jsonpath.MustCompile("$.spec.revisionHistoryLimit").DeleteIfMatch(o.Object, int64(10))

		for _, c := range jsonpath.MustCompile("$.spec.template.spec").Get(o.Object) {
			defaultPodSpec(c.(map[string]interface{}))
		}

		jsonpath.MustCompile("$.spec.updateStrategy.rollingUpdate.maxUnavailable").DeleteIfMatch(o.Object, int64(1))
		jsonpath.MustCompile("$.spec.updateStrategy.rollingUpdate").DeleteIfMatch(o.Object, map[string]interface{}{})
		jsonpath.MustCompile("$.spec.updateStrategy.type").DeleteIfMatch(o.Object, "RollingUpdate")

	case "Deployment.apps":
		jsonpath.MustCompile("$.spec.progressDeadlineSeconds").DeleteIfMatch(o.Object, int64(600))
		jsonpath.MustCompile("$.spec.revisionHistoryLimit").DeleteIfMatch(o.Object, int64(10))

		jsonpath.MustCompile("$.spec.strategy.rollingUpdate.maxSurge").DeleteIfMatch(o.Object, "25%")
		jsonpath.MustCompile("$.spec.strategy.rollingUpdate.maxUnavailable").DeleteIfMatch(o.Object, "25%")
		jsonpath.MustCompile("$.spec.strategy.rollingUpdate").DeleteIfMatch(o.Object, map[string]interface{}{})
		jsonpath.MustCompile("$.spec.strategy.type").DeleteIfMatch(o.Object, "RollingUpdate")

		for _, c := range jsonpath.MustCompile("$.spec.template.spec").Get(o.Object) {
			defaultPodSpec(c.(map[string]interface{}))
		}

	case "Secret":
		jsonpath.MustCompile("$.type").DeleteIfMatch(o.Object, "Opaque")

	case "Service":
		jsonpath.MustCompile("$.spec.ports.*.protocol").DeleteIfMatch(o.Object, "TCP")
		jsonpath.MustCompile("$.spec.sessionAffinity").DeleteIfMatch(o.Object, "None")
		jsonpath.MustCompile("$.spec.type").DeleteIfMatch(o.Object, "ClusterIP")

		for _, p := range jsonpath.MustCompile("$.spec.ports.*").Get(o.Object) {
			jsonpath.MustCompile("$.targetPort").DeleteIfMatch(p, jsonpath.MustCompile("$.port").Get(p)[0].(int64))
		}

	case "StatefulSet.apps":
		jsonpath.MustCompile("$.spec.podManagementPolicy").DeleteIfMatch(o.Object, "OrderedReady")
		jsonpath.MustCompile("$.spec.revisionHistoryLimit").DeleteIfMatch(o.Object, int64(10))
		jsonpath.MustCompile("$.spec.serviceName").DeleteIfMatch(o.Object, "")

		jsonpath.MustCompile("$.spec.updateStrategy.rollingUpdate.partition").DeleteIfMatch(o.Object, int64(0))
		jsonpath.MustCompile("$.spec.updateStrategy.rollingUpdate").DeleteIfMatch(o.Object, map[string]interface{}{})
		jsonpath.MustCompile("$.spec.updateStrategy.type").DeleteIfMatch(o.Object, "RollingUpdate")

		for _, c := range jsonpath.MustCompile("$.spec.template.spec").Get(o.Object) {
			defaultPodSpec(c.(map[string]interface{}))
		}

	case "StorageClass.storage.k8s.io":
		jsonpath.MustCompile("$.reclaimPolicy").DeleteIfMatch(o.Object, "Delete")
		jsonpath.MustCompile("$.volumeBindingMode").DeleteIfMatch(o.Object, "Immediate")
	}
}
