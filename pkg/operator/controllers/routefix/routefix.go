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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/rbac"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	kubeName            = "routefix"
	kubeNamespace       = "openshift-azure-routefix"
	kubeServiceAccount  = "system:serviceaccount:" + kubeNamespace + ":default"
	containerNameDrop   = "drop-icmp"
	containerNameDetect = "detect"
	shellScriptLog      = `while true;
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
	shellScriptDrop = `set -xe
if [[ -f "/env/_master" ]]; then
	set -o allexport
	source "/env/_master"
	set +o allexport
fi

echo "I$(date "+%m%d %H:%M:%S.%N") - drop-icmp - start drop-icmp ${K8S_NODE}"
iptables -X CHECK_ICMP_SOURCE || true
iptables -N CHECK_ICMP_SOURCE || true
iptables -F CHECK_ICMP_SOURCE
iptables -D INPUT -p icmp --icmp-type fragmentation-needed -j CHECK_ICMP_SOURCE || true
iptables -I INPUT -p icmp --icmp-type fragmentation-needed -j CHECK_ICMP_SOURCE
iptables -N ICMP_ACTION || true
iptables -F ICMP_ACTION
iptables -A ICMP_ACTION -j LOG
iptables -A ICMP_ACTION -j DROP
oc observe nodes -a '{ .status.addresses[1].address }' -- /tmp/add_iptables.sh
tail -F /dev/null`
	shellScriptAddIptables = `#!/bin/sh
echo "Adding ICMP drop rule for '$2' " 
#iptables -C CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION || iptables -A CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION
if iptables -C CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION 
then
	echo "iptables already set for $2"
else
	iptables -A CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION
fi
#iptables -nvL 
`
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
	resourceCPU, err1 := resource.ParseQuantity("10m")
	if err1 != nil {
		return nil, err1
	}
	resourceMemory, err2 := resource.ParseQuantity("300Mi")
	if err2 != nil {
		return nil, err2
	}
	defaultMode555 := int32(555)
	return []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        kubeNamespace,
				Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
			},
		},
		scc,
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubeName,
				Namespace: kubeNamespace,
			},
		},
		&rbac.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubeName,
			},
			RoleRef: rbac.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "openshift-sd-controller",
			},
			Subjects: []rbac.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      kubeName,
					Namespace: kubeNamespace,
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubeName,
				Namespace: kubeNamespace,
			},
			Data: map[string]string{
				"add_iptables.sh": shellScriptAddIptables,
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubeName,
				Namespace: kubeNamespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": kubeName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": kubeName},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  containerNameDrop,
								Image: version.RouteFixImage(cluster.Spec.ACRDomain),
								Args: []string{
									"sh",
									"-c",
									shellScriptDrop,
								},
								// TODO: specify requests/limits
								SecurityContext: &corev1.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.Handler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"/bin/bash",
												"-c",
												"echo drop-icmp done",
											},
										},
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "host-slash",
										MountPath: "/",
										ReadOnly:  false,
									},
									{
										Name:      "add-iptables",
										MountPath: "/tmp/add_iptables.sh",
										SubPath:   "add_iptables.sh",
										ReadOnly:  false,
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resourceCPU,
										corev1.ResourceMemory: resourceMemory,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "OVN_KUBE_LOG_LEVEL",
										Value: "4",
									},
									{
										Name: "K8S_NODE",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "spec.nodeName",
											},
										},
									},
								},
							},
							{
								Name:  containerNameDetect,
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
										Name:      "host-slash",
										MountPath: "/",
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
								Name: "host-slash",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/",
										Type: &hostPathUnset,
									},
								},
							},
							{
								Name: "add-iptables",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "add-iptables",
										},
										DefaultMode: &defaultMode555,
									},
								},
							},
						},
						DNSPolicy:     corev1.DNSClusterFirst,
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
		},
	}, nil
}
