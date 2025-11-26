package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	_ "embed"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type degradedEtcd struct {
	Node  string
	Pod   string
	NewIP string
	OldIP string
}

type podFailedError struct {
	podName string
	phase   corev1.PodPhase
	message string
}

func (e *podFailedError) Error() string {
	return fmt.Sprintf("pod %s event %s received with message %s", e.podName, e.phase, e.message)
}

const (
	serviceAccountName    = "etcd-recovery-privileged"
	kubeServiceAccount    = "system:serviceaccount" + namespaceEtcds + ":" + serviceAccountName
	namespaceEtcds        = "openshift-etcd"
	image                 = "ubi9/ubi-minimal"
	genericPodName        = "etcd-recovery-"
	patchOverides         = "unsupportedConfigOverrides:"
	patchDisableOverrides = `{"useUnsupportedUnsafeNonHANonProductionUnstableEtcd": true}`
)

// fixEtcd performs a single master node etcd recovery based on these steps and scenarios:
// https://docs.openshift.com/container-platform/4.10/backup_and_restore/control_plane_backup_and_restore/replacing-unhealthy-etcd-member.html
func (f *frontend) fixEtcd(ctx context.Context, log *logrus.Entry, env env.Interface, doc *api.OpenShiftClusterDocument, kubeActions adminactions.KubeActions, etcdcli operatorv1client.EtcdInterface) ([]byte, error) {
	log.Info("Starting Etcd Recovery now")

	log.Infof("Listing etcd pods now")
	rawPods, err := kubeActions.KubeList(ctx, "Pod", namespaceEtcds)
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	pods := &corev1.PodList{}
	err = codec.NewDecoderBytes(rawPods, &codec.JsonHandle{}).Decode(pods)
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode pods, %s", err.Error()))
	}

	de, err := findDegradedEtcd(log, pods)
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	log.Infof("Found degraded endpoint: %v", de)

	backupContainerLogs, err := backupEtcdData(ctx, log, de.Node, kubeActions)
	if err != nil {
		return backupContainerLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	fixPeersContainerLogs, err := fixPeers(ctx, log, de, pods, kubeActions, doc.OpenShiftCluster.Name)
	allLogs, _ := logSeperator(backupContainerLogs, fixPeersContainerLogs)
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	rawEtcd, err := kubeActions.KubeGet(ctx, "Etcd", "", "cluster")
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	log.Info("Getting etcd operating now")
	etcd := &operatorv1.Etcd{}
	err = codec.NewDecoderBytes(rawEtcd, &codec.JsonHandle{}).Decode(etcd)
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode etcd operator, %s", err.Error()))
	}

	existingOverrides := etcd.Spec.UnsupportedConfigOverrides.Raw
	etcd.Spec.UnsupportedConfigOverrides = kruntime.RawExtension{
		Raw: []byte(patchDisableOverrides),
	}
	err = patchEtcd(ctx, log, etcdcli, etcd, patchDisableOverrides)
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	err = deleteSecrets(ctx, log, kubeActions, de, doc.OpenShiftCluster.Properties.InfraID)
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	etcd.Spec.ForceRedeploymentReason = fmt.Sprintf("single-master-recovery-%s", time.Now())
	err = patchEtcd(ctx, log, etcdcli, etcd, etcd.Spec.ForceRedeploymentReason)
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	etcd.Spec.UnsupportedConfigOverrides.Raw = existingOverrides
	err = patchEtcd(ctx, log, etcdcli, etcd, patchOverides+string(etcd.Spec.UnsupportedConfigOverrides.Raw))
	if err != nil {
		return allLogs, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return allLogs, nil
}

func logSeperator(log1, log2 []byte) ([]byte, error) {
	logSeperator := "\n" + strings.Repeat("#", 150) + "\n"
	allLogs := append(log1, []byte(logSeperator)...)
	allLogs = append(allLogs, log2...)

	buf := &bytes.Buffer{}
	return buf.Bytes(), codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(allLogs)
}

// patchEtcd patches the etcd object provided and logs the patch string
func patchEtcd(ctx context.Context, log *logrus.Entry, etcdcli operatorv1client.EtcdInterface, e *operatorv1.Etcd, patch string) error {
	log.Infof("Preparing to patch etcd %s with %s", e.Name, patch)
	// must be removed to force redeployment
	e.CreationTimestamp = metav1.Time{
		Time: time.Now(),
	}
	e.ResourceVersion = ""
	e.UID = ""

	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(e)
	if err != nil {
		return err
	}
	_, err = etcdcli.Patch(ctx, e.Name, types.MergePatchType, buf.Bytes(), metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Infof("Patched etcd %s with %s", e.Name, patch)

	return nil
}

func deleteSecrets(ctx context.Context, log *logrus.Entry, kubeActions adminactions.KubeActions, de *degradedEtcd, infraID string) error {
	for _, prefix := range []string{"etcd-peer-", "etcd-serving-", "etcd-serving-metrics-"} {
		secret := prefix + de.Node
		log.Infof("Deleting secret %s", secret)
		err := kubeActions.KubeDelete(ctx, "Secret", namespaceEtcds, secret, false, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func getPeerPods(pods []corev1.Pod, de *degradedEtcd, cluster string) (string, error) {
	regNode, err := regexp.Compile(".master-[0-9]$")
	if err != nil {
		return "", err
	}
	regPod, err := regexp.Compile("etcd-" + cluster + "-[0-9A-Za-z]*-master-[0-9]$")
	if err != nil {
		return "", err
	}

	var peerPods string
	for _, p := range pods {
		if regNode.MatchString(p.Spec.NodeName) &&
			regPod.MatchString(p.Name) &&
			p.Name != de.Pod {
			peerPods += p.Name + " "
		}
	}
	return peerPods, nil
}

func newPodFixPeers(peerPods, deNode string) (*unstructured.Unstructured, error) {
	const podNameFixPeers = genericPodName + "fix-peers"

	podManifest := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podNameFixPeers,
			Namespace: namespaceEtcds,
			Labels:    map[string]string{"app": podNameFixPeers},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:      corev1.RestartPolicyNever, // It's safer to let the pod and the geneva action to fail once than let it constantly retry.
			ServiceAccountName: serviceAccountName,
			Containers: []corev1.Container{
				{
					Name:  podNameFixPeers,
					Image: image,
					Command: []string{
						"/bin/bash",
						"-cx",
						backupOrFixEtcd,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: pointerutils.ToPtr(true),
					},
					Env: []corev1.EnvVar{
						{
							Name:  "PEER_PODS",
							Value: peerPods,
						},
						{
							Name:  "DEGRADED_NODE",
							Value: deNode,
						},
						{
							Name:  "FIX_PEERS",
							Value: "true",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "host",
							MountPath: "/host",
							ReadOnly:  false,
						},
					},
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
			},
		},
	}
	// Frontend kubeactions expects an unstructured type
	unstructuredPod, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(&podManifest)

	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: unstructuredPod,
	}, nil
}

// fixPeers creates a pod that ssh's into the failing pod's peer pods, and deletes the failing etcd member from it's member's list
func fixPeers(ctx context.Context, log *logrus.Entry, de *degradedEtcd, pods *corev1.PodList, kubeActions adminactions.KubeActions, cluster string) ([]byte, error) {
	peerPods, err := getPeerPods(pods.Items, de, cluster)
	if err != nil {
		return []byte{}, err
	}

	podFixPeers, err := newPodFixPeers(peerPods, de.Node)
	if err != nil {
		return []byte{}, err
	}

	cleanup, err, nestedCleanupErr := createPrivilegedServiceAccount(ctx, log, serviceAccountName, cluster, kubeServiceAccount, kubeActions)
	if err != nil {
		return []byte{}, err
	}
	if nestedCleanupErr != nil {
		return []byte{}, nestedCleanupErr
	}
	defer func() {
		if cleanup != nil {
			if err := cleanup(); err != nil {
				log.WithError(err).Error("error cleaning up")
			}
		}
	}()

	log.Infof("Creating pod %s", podFixPeers.GetName())
	err = kubeActions.KubeCreateOrUpdate(ctx, podFixPeers)
	if err != nil {
		return []byte{}, err
	}

	watcher, err := kubeActions.KubeWatch(ctx, podFixPeers, "app")
	if err != nil {
		return []byte{}, err
	}

	containerLogs, err := waitAndGetPodLogs(ctx, log, watcher, podFixPeers, kubeActions)
	if err != nil {
		return containerLogs, err
	}

	log.Infof("Deleting %s now", podFixPeers.GetName())
	propPolicy := metav1.DeletePropagationBackground
	err = kubeActions.KubeDelete(ctx, podFixPeers.GetKind(), namespaceEtcds, podFixPeers.GetName(), true, &propPolicy)
	if err != nil {
		return containerLogs, err
	}

	// return errors from deferred delete functions
	return containerLogs, err
}

func newServiceAccount(name, cluster string) *unstructured.Unstructured {
	serviceAcc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"automountServiceAccountToken": pointerutils.ToPtr(true),
		},
	}
	serviceAcc.SetAPIVersion("v1")
	serviceAcc.SetKind("ServiceAccount")
	serviceAcc.SetName(name)
	serviceAcc.SetNamespace(namespaceEtcds)

	return serviceAcc
}

func newClusterRole(usersAccount, cluster string) *unstructured.Unstructured {
	clusterRole := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"rules": []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get", "create"},
					Resources: []string{"pods", "pods/exec"},
					APIGroups: []string{""},
				},
			},
		},
	}
	// Cluster Role isn't scoped to a namespace
	clusterRole.SetAPIVersion("rbac.authorization.k8s.io/v1")
	clusterRole.SetKind("ClusterRole")
	clusterRole.SetName(usersAccount)

	return clusterRole
}

func newClusterRoleBinding(name, cluster string) *unstructured.Unstructured {
	crb := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"roleRef": map[string]interface{}{
				"kind":      "ClusterRole",
				"name":      kubeServiceAccount,
				"apiGroups": "",
			},
			"subjects": []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      name,
					Namespace: namespaceEtcds,
				},
			},
		},
	}
	crb.SetAPIVersion("rbac.authorization.k8s.io/v1")
	crb.SetKind("ClusterRoleBinding")
	crb.SetName(name)

	return crb
}

func newSecurityContextConstraint(name, cluster, usersAccount string) *unstructured.Unstructured {
	scc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"groups":                   []string{},
			"users":                    []string{usersAccount},
			"allowPrivilegedContainer": true,
			"allowPrivilegeEscalation": pointerutils.ToPtr(true),
			"allowedCapabilities":      []corev1.Capability{"*"},
			"runAsUser": map[string]securityv1.RunAsUserStrategyType{
				"type": securityv1.RunAsUserStrategyRunAsAny,
			},
			"seLinuxContext": map[string]securityv1.SELinuxContextStrategyType{
				"type": securityv1.SELinuxStrategyRunAsAny,
			},
		},
	}
	scc.SetAPIVersion("security.openshift.io/v1")
	scc.SetKind("SecurityContextConstraints")
	scc.SetName(name)

	return scc
}

// createPrivilegedServiceAccount creates the following objects and returns a cleanup function to delete them all after use
//
// - ServiceAccount
//
// - ClusterRole
//
// - ClusterRoleBinding
//
// - SecurityContextConstraint
func createPrivilegedServiceAccount(ctx context.Context, log *logrus.Entry, name, cluster, usersAccount string, kubeActions adminactions.KubeActions) (func() error, error, error) {
	serviceAcc := newServiceAccount(name, cluster)
	clusterRole := newClusterRole(usersAccount, cluster)
	crb := newClusterRoleBinding(name, cluster)
	scc := newSecurityContextConstraint(name, cluster, usersAccount)

	// cleanup is created here incase an error occurs while creating permissions
	cleanup := func() error {
		log.Infof("Deleting service account %s now", serviceAcc.GetName())
		err := kubeActions.KubeDelete(ctx, serviceAcc.GetKind(), serviceAcc.GetNamespace(), serviceAcc.GetName(), true, nil)
		if err != nil {
			return err
		}

		log.Infof("Deleting security context contstraint %s now", scc.GetName())
		err = kubeActions.KubeDelete(ctx, scc.GetKind(), scc.GetNamespace(), scc.GetName(), true, nil)
		if err != nil {
			return err
		}

		log.Infof("Deleting cluster role %s now", clusterRole.GetName())
		err = kubeActions.KubeDelete(ctx, clusterRole.GetKind(), clusterRole.GetNamespace(), clusterRole.GetName(), true, nil)
		if err != nil {
			return err
		}

		log.Infof("Deleting cluster role binding %s now", crb.GetName())
		err = kubeActions.KubeDelete(ctx, crb.GetKind(), crb.GetNamespace(), crb.GetName(), true, nil)
		if err != nil {
			return err
		}

		return nil
	}

	log.Infof("Creating Service Account %s now", serviceAcc.GetName())
	err := kubeActions.KubeCreateOrUpdate(ctx, serviceAcc)
	if err != nil {
		return nil, err, cleanup()
	}

	log.Infof("Creating Cluster Role %s now", clusterRole.GetName())
	err = kubeActions.KubeCreateOrUpdate(ctx, clusterRole)
	if err != nil {
		return nil, err, cleanup()
	}

	log.Infof("Creating Cluster Role Binding %s now", crb.GetName())
	err = kubeActions.KubeCreateOrUpdate(ctx, crb)
	if err != nil {
		return nil, err, cleanup()
	}

	log.Infof("Creating Security Context Constraint %s now", name)
	err = kubeActions.KubeCreateOrUpdate(ctx, scc)
	if err != nil {
		return nil, err, cleanup()
	}

	return cleanup, nil, nil
}

// backupEtcdData creates a job that creates two backups on the node
//
// /etc/kubernetes/manifests/etcd-pod.yaml is moved to /var/lib/etcd-backup/etcd-pod.yaml
// the purpose of this is to stop the failing etcd pod from crashlooping by removing the manifest
//
// The second backup
// /var/lib/etcd is moved to /tmp
//
// If backups already exists the job is cowardly and refuses to overwrite them
func backupEtcdData(ctx context.Context, log *logrus.Entry, node string, kubeActions adminactions.KubeActions) ([]byte, error) {
	podDataBackup, err := createBackupEtcdDataPod(node)

	if err != nil {
		return []byte{}, err
	}

	log.Infof("Creating pod %s", podDataBackup.GetName())
	err = kubeActions.KubeCreateOrUpdate(ctx, podDataBackup)
	if err != nil {
		return []byte{}, err
	}
	log.Infof("Pod %s has been created", podDataBackup.GetName())

	watcher, err := kubeActions.KubeWatch(ctx, podDataBackup, "app")
	if err != nil {
		return []byte{}, err
	}
	containerLogs, err := waitAndGetPodLogs(ctx, log, watcher, podDataBackup, kubeActions)
	if err != nil {
		return containerLogs, err
	}

	log.Infof("Deleting pod %s now", podDataBackup.GetName())
	propPolicy := metav1.DeletePropagationBackground
	return containerLogs, kubeActions.KubeDelete(ctx, podDataBackup.GetKind(), namespaceEtcds, podDataBackup.GetName(), true, &propPolicy)
}

func waitForPodSucceed(ctx context.Context, log *logrus.Entry, watcher watch.Interface) error {
	var pod corev1.Pod

	for {
		select {
		case event := <-watcher.ResultChan():
			log.Infoln("Event received")

			u, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				return fmt.Errorf("unexpected event: %#v", event)
			}
			err := kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &pod)
			if err != nil {
				return fmt.Errorf("failed to convert unstructured object to Pod: %w", err)
			}
			log.Infof("Pod Status Phase %s", pod.Status.Phase)

			switch pod.Status.Phase {
			case corev1.PodSucceeded:
				log.Infof("Pod %s completed with %s: %s", pod.GetName(), pod.Status.Message, pod.Status.Phase)
				return nil

			case corev1.PodFailed:
				log.Infof("Pod %s reached phase %s with message: %s", pod.GetName(), pod.Status.Phase, pod.Status.Message)
				return &podFailedError{
					podName: pod.GetName(),
					phase:   pod.Status.Phase,
					message: pod.Status.Message,
				}
			}

		case <-ctx.Done():
			return fmt.Errorf("context was cancelled while waiting for pod to succeed because %s", ctx.Err())
		}
	}
}

// Waits until a standalone pod succeeds or fails. There is no retry logic for failures.
func waitAndGetPodLogs(ctx context.Context, log *logrus.Entry, watcher watch.Interface, o *unstructured.Unstructured, k adminactions.KubeActions) ([]byte, error) {
	var waitErr error
	var originalPod corev1.Pod

	log.Infof("Waiting for %s to reach %s phase", o.GetName(), corev1.PodSucceeded)
	err := waitForPodSucceed(ctx, log, watcher)

	if err != nil {
		var failedErr *podFailedError
		if !errors.As(err, &failedErr) {
			return nil, err
		}
		waitErr = err // waitForPodSucceed returned a pod failed error. We want to keep this and get the pod logs.
	}

	// get container spec
	err = kruntime.DefaultUnstructuredConverter.FromUnstructured(o.UnstructuredContent(), &originalPod)
	if err != nil {
		// We should get a corev1.Pod struct here, report failure if we don't
		return nil, err
	}

	if len(originalPod.Spec.Containers) != 1 {
		// we should have only one container in the original pod spec
		return nil, fmt.Errorf("unexpected number of containers in %v. There are %d containers, but only one container should be in the spec", originalPod.Name, len(originalPod.Spec.Containers))
	}

	cxName := originalPod.Spec.Containers[0].Name
	log.Infof("Collecting container logs for Pod %s, container %s, in namespace %s", originalPod.Name, cxName, originalPod.Namespace)

	cxLogs, err := k.KubeGetPodLogs(ctx, originalPod.Namespace, originalPod.Name, cxName)
	if err != nil {
		return cxLogs, err
	}
	log.Infof("Successfully collected logs for %s", originalPod.Name)

	return cxLogs, waitErr
}

func createBackupEtcdDataPod(node string) (*unstructured.Unstructured, error) {
	const podNameDataBackup = genericPodName + "data-backup"
	podManifest := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podNameDataBackup,
			Namespace: namespaceEtcds,
			Labels:    map[string]string{"app": podNameDataBackup},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever, // It's safer to let the pod and the geneva action to fail once than let it constantly retry.
			NodeName:      node,
			Containers: []corev1.Container{
				{
					Name:  podNameDataBackup,
					Image: image,
					Command: []string{
						"chroot",
						"/host",
						"/bin/bash",
						"-c",
						backupOrFixEtcd,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "host",
							MountPath: "/host",
							ReadOnly:  false,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{"SYS_CHROOT"},
						},
						Privileged: pointerutils.ToPtr(true),
					},
					Env: []corev1.EnvVar{
						{
							Name:  "BACKUP",
							Value: "true",
						},
					},
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
			},
		},
	}

	// Frontend kubeactions expects an unstructured type
	unstructuredPod, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(&podManifest)

	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: unstructuredPod,
	}, nil
}

func comparePodEnvToIp(log *logrus.Entry, pods *corev1.PodList) (*degradedEtcd, error) {
	degradedEtcds := []degradedEtcd{}
	for _, p := range pods.Items {
		envIP := ipFromEnv(p.Spec.Containers, p.Name)
		for _, podIP := range p.Status.PodIPs {
			if podIP.IP != envIP && envIP != "" {
				log.Infof("Found conflicting IPs for etcd Pod %s: Pod IP: %s != ENV IP %s", p.Name, podIP.IP, envIP)
				degradedEtcds = append(degradedEtcds, degradedEtcd{
					Node:  strings.ReplaceAll(p.Name, "etcd-", ""),
					Pod:   p.Name,
					NewIP: podIP.IP,
					OldIP: envIP,
				})
				break
			}
		}
	}

	// Check for multiple etcd pods with IP address conflicts
	var de *degradedEtcd
	if len(degradedEtcds) > 1 {
		return nil, fmt.Errorf("found multiple etcd pods with conflicting IP addresses, only one degraded etcd is supported, unable to recover. Conflicting IPs found: %v", degradedEtcds)
		// happens if the env variables are empty, check statuses next
	} else if len(degradedEtcds) == 0 {
		de = &degradedEtcd{}
	} else {
		// array is no longer needed
		de = &degradedEtcds[0]
	}

	return de, nil
}

// comparePodEnvToIp compares the etcd container's environment variables to the pod's actual IP address
func findDegradedEtcd(log *logrus.Entry, pods *corev1.PodList) (*degradedEtcd, error) {
	de, err := comparePodEnvToIp(log, pods)
	if err != nil {
		return &degradedEtcd{}, err
	}

	crashingPodSearchDe, err := findCrashloopingPods(log, pods)
	log.Infof("Found degraded etcd while searching by Pod statuses: %v", crashingPodSearchDe)
	if err != nil {
		return &degradedEtcd{}, err
	}

	// Sanity check
	// Since we are checking for both an etcd Pod with an IP mis match, and the statuses of all etcd pods, let's make sure the Pod's returned by both are the same
	if de.Pod != crashingPodSearchDe.Pod && de.Pod != "" {
		return de, fmt.Errorf("etcd Pod found in crashlooping state %s is not equal to etcd Pod with IP ENV mis match %s... failed sanity check", de.Pod, crashingPodSearchDe.Pod)
	}

	// If no conflict is found a recent IP change may still be causing an issue
	// Sometimes etcd can recovery the deployment itself, however there is still a data directory with the previous member's IP address present causing a failure
	// This can still be remediated by relying on the pod statuses
	if de.Node == "" {
		log.Info("Unable to find an IP address conflict, using etcd Pod found during search by statuses")
		return crashingPodSearchDe, nil
	}

	return de, nil
}

func ipFromEnv(containers []corev1.Container, podName string) string {
	for _, c := range containers {
		if c.Name == "etcd" {
			for _, e := range c.Env {
				// The environment variable that contains etcd's IP address has the following naming convention
				// NODE_cluster_name_infra_ID_master_0_IP
				// while the pod looks like this
				// etcd-cluster-name-infra-id-master-0
				// To find the pod's IP address by variable name we use the pod's name
				envName := strings.ReplaceAll(strings.ReplaceAll(podName, "-", "_"), "etcd_", "NODE_")
				if e.Name == fmt.Sprintf("%s_IP", envName) {
					return e.Value
				}
			}
		}
	}

	return ""
}

func findCrashloopingPods(log *logrus.Entry, pods *corev1.PodList) (*degradedEtcd, error) {
	// pods are collected in a list to check for multiple crashing etcd instances
	// multiple etcd failures aren't supported so an error will be returned, rather than assuming the first found is the only one
	crashingPods := &corev1.PodList{}
	for _, p := range pods.Items {
		for _, c := range p.Status.ContainerStatuses {
			if !c.Ready && c.Name == "etcd" {
				log.Infof("Found etcd container with status: %v", c)
				crashingPods.Items = append(crashingPods.Items, p)
			}
		}
	}

	if len(crashingPods.Items) > 1 {
		// log multiple names in a readable way
		names := []string{}
		for _, c := range crashingPods.Items {
			names = append(names, c.Name)
		}
		return nil, fmt.Errorf("only a single degraded etcd pod can can be recovered from, more than one NotReady etcd pods were found: %v", names)
	} else if len(crashingPods.Items) == 0 {
		return nil, errors.New("no etcd pod's were found in a CrashLoopBackOff state, unable to remediate etcd deployment")
	}
	crashingPod := &crashingPods.Items[0]

	return &degradedEtcd{
		Node:  strings.ReplaceAll(crashingPod.Name, "etcd-", ""),
		Pod:   crashingPod.Name,
		OldIP: "unknown",
		NewIP: "unknown",
	}, nil
}
