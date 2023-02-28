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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	// image is the openshift-sdn image from 4.6.18
	image               = "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:bcc1fb20f06f00829727cb46ff21e22103fd4c737fdcbbf2fab13121f31ebcbd"
	kubeName            = "routefix"
	serviceAccountName  = "routefix"
	kubeNamespace       = "openshift-azure-routefix"
	kubeServiceAccount  = "system:serviceaccount:" + kubeNamespace + ":" + serviceAccountName
	configmapName       = "add-iptables"
	configmapScriptName = "add_iptables.sh"
	configmapScriptDir  = "/tmp"

	shellScriptLog = `while true;
do
	NOW=$(date "+%Y-%m-%d %H:%M:%S")
	DROPPED_PACKETS=$(ovs-ofctl -O OpenFlow13 dump-flows unix:/host/var/run/openvswitch/br0.mgmt | sed -ne '/table=10,.* actions=drop/ { s/.* n_packets=//; s/,.*//; p }')
	if [ "$DROPPED_PACKETS" != "" ] && [ "$DROPPED_PACKETS" -gt 1000 ];
	then
		echo "$NOW table=10 actions=drop packets=$DROPPED_PACKETS broken=true"
	else
		echo "$NOW table=10 actions=drop packets=$DROPPED_PACKETS broken=false"
	fi
	sleep 60
done`

	shellScriptDrop = `set -xe
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
/host/usr/bin/oc observe nodes -a '{ .status.addresses[1].address }' -- ` + configmapScriptDir + `/` + configmapScriptName

	shellScriptAddIptables = `#!/bin/sh
echo "Adding ICMP drop rule for '$2' "
#iptables -C CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION || iptables -A CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION
if iptables -C CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION
then
	echo "iptables already set for $2"
else
	iptables -A CHECK_ICMP_SOURCE -p icmp -s $2 -j ICMP_ACTION
fi
#iptables -nvL`
)

func (r *Reconciler) securityContextConstraints(ctx context.Context, name, serviceAccountName string) (*securityv1.SecurityContextConstraints, error) {
	scc := &securityv1.SecurityContextConstraints{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "privileged"}, scc)
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

func (r *Reconciler) resources(ctx context.Context, cluster *arov1alpha1.Cluster) ([]kruntime.Object, error) {
	scc, err := r.securityContextConstraints(ctx, "privileged-routefix", kubeServiceAccount)
	if err != nil {
		return nil, err
	}
	resourceCPU, err := resource.ParseQuantity("10m")
	if err != nil {
		return nil, err
	}
	resourceMemory, err := resource.ParseQuantity("300Mi")
	if err != nil {
		return nil, err
	}

	return []kruntime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        kubeNamespace,
				Annotations: map[string]string{projectv1.ProjectNodeSelector: ""},
			},
		},
		scc,
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: kubeNamespace,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubeName,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "openshift-sdn-controller",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: kubeNamespace,
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configmapName,
				Namespace: kubeNamespace,
			},
			Data: map[string]string{
				configmapScriptName: shellScriptAddIptables,
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
						ServiceAccountName: serviceAccountName,
						Containers: []corev1.Container{
							{
								Name:  "drop-icmp",
								Image: image,
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
									PreStop: &corev1.LifecycleHandler{
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
										Name:      "host",
										MountPath: "/host",
										ReadOnly:  false,
									},
									{
										Name:      configmapName,
										MountPath: configmapScriptDir + "/" + configmapScriptName,
										SubPath:   configmapScriptName,
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
								Name:  "detect",
								Image: image,
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
										Name:      "host",
										MountPath: "/host",
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
								Name: "host",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/",
									},
								},
							},
							{
								Name: configmapName,
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: configmapName,
										},
										DefaultMode: to.Int32Ptr(0555),
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
