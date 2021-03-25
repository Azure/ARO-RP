package routefix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	projectv1 "github.com/openshift/api/project/v1"
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	kubeName           = "routefix"
	kubeNameLog        = "routefix-log"
	kubeNamespace      = "openshift-azure-routefix"
	kubeServiceAccount = "system:serviceaccount:" + kubeNamespace + ":default"
	shellScript        = `for ((;;))
do
  if ip route show cache | grep -q 'mtu 1450'; then
    ip route show cache
    ip route flush cache
  fi
  sleep 60
done`
	shellScriptLog = `while true;
do
	NOW=$(date "+%Y-%m-%d %H:%M:%S")
	DROPPED_PACKETS=$(ovs-ofctl -O OpenFlow13 dump-flows br0 | sed -ne '/table=10,.* actions=drop/ { s/.* n_packets=//; s/,.*//; p }')
	if [ "$DROPPED_PACKETS" != "" ] && [ "$DROPPED_PACKETS" -gt 1000 ];
	then
		echo "$NOW table=10 actions=drop packets=$DROPPED_PACKETS broken=true"
	else
		echo "$NOW table=10 actions=drop packets=$DROPPED_PACKETS broken=false"
	fi
	sleep 60
done`
)

func (r *RouteFixReconciler) securityContextConstraints(ctx context.Context, name, serviceAccountName string) (*securityv1.SecurityContextConstraints, error) {
	scc, err := r.securitycli.SecurityV1().SecurityContextConstraints().Get(ctx, "privileged", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	scc.ObjectMeta = metav1.ObjectMeta{
		Name: name,
	}
	scc.Groups = []string{}
	scc.Users = []string{serviceAccountName}
	return scc, nil
}

func (r *RouteFixReconciler) resources(ctx context.Context, cluster *arov1alpha1.Cluster) ([]runtime.Object, error) {
	scc, err := r.securityContextConstraints(ctx, "privileged-routefix", kubeServiceAccount)
	if err != nil {
		return nil, err
	}
	hostPathUnset := corev1.HostPathUnset
	return []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        kubeNamespace,
				Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
			},
		},
		scc,
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubeName,
				Namespace: kubeNamespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "routefix"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "routefix"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  kubeName,
								Image: version.RouteFixImage(cluster.Spec.ACRDomain),
								Args: []string{
									"sh",
									"-c",
									shellScript,
								},
								// TODO: specify requests/limits
								SecurityContext: &corev1.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
							{
								Name:  kubeNameLog,
								Image: version.RouteFixImage(cluster.Spec.ACRDomain),
								Args: []string{
									"sh",
									"-c",
									shellScriptLog,
								},
								// TODO: specify requests/limits
								SecurityContext: &corev1.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "host-run",
										MountPath: "/run",
										ReadOnly:  true,
									},
								},
							},
						},
						HostNetwork: true,
						Tolerations: []corev1.Toleration{
							{
								Effect:   corev1.TaintEffectNoExecute,
								Operator: corev1.TolerationOpExists,
							},
							{
								Effect:   corev1.TaintEffectNoSchedule,
								Operator: corev1.TolerationOpExists,
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "host-run",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/run",
										Type: &hostPathUnset,
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}
