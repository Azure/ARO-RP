package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
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
)

type degradedEtcd struct {
	Node  string
	Pod   string
	NewIP string
	OldIP string
}

const (
	serviceAccountName    = "etcd-recovery-privileged"
	kubeServiceAccount    = "system:serviceaccount" + namespaceEtcds + ":" + serviceAccountName
	namespaceEtcds        = "openshift-etcd"
	image                 = "ubi9/ubi-minimal"
	jobName               = "etcd-recovery-"
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

	backupContainerLogs, err := backupEtcdData(ctx, log, doc.OpenShiftCluster.Name, de.Node, kubeActions)
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

	etcd.Spec.OperatorSpec.UnsupportedConfigOverrides.Raw = existingOverrides
	err = patchEtcd(ctx, log, etcdcli, etcd, patchOverides+string(etcd.Spec.OperatorSpec.UnsupportedConfigOverrides.Raw))
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

func newJobFixPeers(cluster, peerPods, deNode string) *unstructured.Unstructured {
	const jobNameFixPeers = jobName + "fix-peers"
	// Frontend kubeactions expects an unstructured type
	jobFixPeers := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"objectMeta": map[string]interface{}{
				"name":      jobNameFixPeers,
				"namespace": namespaceEtcds,
				"labels":    map[string]string{"app": jobNameFixPeers},
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"objectMeta": map[string]interface{}{
						"name":      jobNameFixPeers,
						"namespace": namespaceEtcds,
						"labels":    map[string]string{"app": jobNameFixPeers},
					},
					"activeDeadlineSeconds":   to.Int64Ptr(10),
					"completions":             to.Int32Ptr(1),
					"ttlSecondsAfterFinished": to.Int32Ptr(300),
					"spec": map[string]interface{}{
						"restartPolicy":      corev1.RestartPolicyOnFailure,
						"serviceAccountName": serviceAccountName,
						"containers": []corev1.Container{
							{
								Name:  jobNameFixPeers,
								Image: image,
								Command: []string{
									"/bin/bash",
									"-cx",
									backupOrFixEtcd,
								},
								SecurityContext: &corev1.SecurityContext{
									Privileged: to.BoolPtr(true),
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
						"volumes": []corev1.Volume{
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
				},
			},
		},
	}

	// This creates an embedded "metadata" map[string]string{} in the unstructured object
	// For an unknown reason, creating "metadata" directly in the object doesn't work
	// and the helper functions must be used
	jobFixPeers.SetKind("Job")
	jobFixPeers.SetAPIVersion("batch/v1")
	jobFixPeers.SetName(jobNameFixPeers)
	jobFixPeers.SetNamespace(namespaceEtcds)

	return jobFixPeers
}

// fixPeers creates a job that ssh's into the failing pod's peer pods, and deletes the failing pod from it's member's list
func fixPeers(ctx context.Context, log *logrus.Entry, de *degradedEtcd, pods *corev1.PodList, kubeActions adminactions.KubeActions, cluster string) ([]byte, error) {
	peerPods, err := getPeerPods(pods.Items, de, cluster)
	if err != nil {
		return []byte{}, err
	}

	jobFixPeers := newJobFixPeers(cluster, peerPods, de.Node)

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

	log.Infof("Creating job %s", jobFixPeers.GetName())
	err = kubeActions.KubeCreateOrUpdate(ctx, jobFixPeers)
	if err != nil {
		return []byte{}, err
	}

	watcher, err := kubeActions.KubeWatch(ctx, jobFixPeers, "app")
	if err != nil {
		return []byte{}, err
	}

	containerLogs, err := waitForJobSucceed(ctx, log, watcher, jobFixPeers, kubeActions)
	if err != nil {
		return containerLogs, err
	}

	log.Infof("Deleting %s now", jobFixPeers.GetName())
	propPolicy := metav1.DeletePropagationBackground
	err = kubeActions.KubeDelete(ctx, "Job", namespaceEtcds, jobFixPeers.GetName(), true, &propPolicy)
	if err != nil {
		return containerLogs, err
	}

	// return errors from deferred delete functions
	return containerLogs, err
}

func newServiceAccount(name, cluster string) *unstructured.Unstructured {
	serviceAcc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"automountServiceAccountToken": to.BoolPtr(true),
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
			"allowPrivilegeEscalation": to.BoolPtr(true),
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
func backupEtcdData(ctx context.Context, log *logrus.Entry, cluster, node string, kubeActions adminactions.KubeActions) ([]byte, error) {
	jobDataBackup := createBackupEtcdDataJob(cluster, node)

	log.Infof("Creating job %s", jobDataBackup.GetName())
	err := kubeActions.KubeCreateOrUpdate(ctx, jobDataBackup)
	if err != nil {
		return []byte{}, err
	}
	log.Infof("Job %s has been created", jobDataBackup.GetName())

	watcher, err := kubeActions.KubeWatch(ctx, jobDataBackup, "app")
	if err != nil {
		return []byte{}, err
	}

	containerLogs, err := waitForJobSucceed(ctx, log, watcher, jobDataBackup, kubeActions)
	if err != nil {
		return containerLogs, err
	}

	log.Infof("Deleting job %s now", jobDataBackup.GetName())
	propPolicy := metav1.DeletePropagationBackground
	return containerLogs, kubeActions.KubeDelete(ctx, "Job", namespaceEtcds, jobDataBackup.GetName(), true, &propPolicy)
}

func waitForJobSucceed(ctx context.Context, log *logrus.Entry, watcher watch.Interface, o *unstructured.Unstructured, k adminactions.KubeActions) ([]byte, error) {
	var waitErr error
	log.Infof("Waiting for %s to reach %s phase", o.GetName(), corev1.PodSucceeded)
	select {
	case event := <-watcher.ResultChan():
		pod := event.Object.(*corev1.Pod)

		if pod.Status.Phase == corev1.PodSucceeded {
			log.Infof("Job %s completed with %s", pod.GetName(), pod.Status.Message)
		} else if pod.Status.Phase == corev1.PodFailed {
			log.Infof("Job %s reached phase %s with message: %s", pod.GetName(), pod.Status.Phase, pod.Status.Message)
			waitErr = fmt.Errorf("pod %s event %s received with message %s", pod.Name, pod.Status.Phase, pod.Status.Message)
		}
	case <-ctx.Done():
		waitErr = fmt.Errorf("context was cancelled while waiting for %s because %s", o.GetName(), ctx.Err())
	}

	// get container name
	cxName := o.UnstructuredContent()["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]corev1.Container)[0].Name
	log.Infof("Collecting container logs for Pod %s, container %s, in namespace %s", o.GetName(), cxName, o.GetNamespace())

	cxLogs, err := k.KubeGetPodLogs(ctx, o.GetNamespace(), o.GetName(), cxName)
	if err != nil {
		return cxLogs, err
	}
	log.Infof("Successfully collected logs for %s", o.GetName())

	return cxLogs, waitErr
}

func createBackupEtcdDataJob(cluster, node string) *unstructured.Unstructured {
	const jobNameDataBackup = jobName + "data-backup"
	j := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"objectMeta": map[string]interface{}{
				"name":      jobNameDataBackup,
				"kind":      "Job",
				"namespace": namespaceEtcds,
				"labels":    map[string]string{"app": jobNameDataBackup},
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"objectMeta": map[string]interface{}{
						"name":      jobNameDataBackup,
						"namespace": namespaceEtcds,
						"labels":    map[string]string{"app": jobNameDataBackup},
					},
					"activeDeadlineSeconds":   to.Int64Ptr(10),
					"completions":             to.Int32Ptr(1),
					"ttlSecondsAfterFinished": to.Int32Ptr(300),
					"spec": map[string]interface{}{
						"restartPolicy": corev1.RestartPolicyOnFailure,
						"nodeName":      node,
						"containers": []corev1.Container{
							{
								Name:  jobNameDataBackup,
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
									Privileged: to.BoolPtr(true),
								},
								Env: []corev1.EnvVar{
									{
										Name:  "BACKUP",
										Value: "true",
									},
								},
							},
						},
						"volumes": []corev1.Volume{
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
				},
			},
		},
	}

	// This creates an embedded "metadata" map[string]string{} in the unstructured object
	// For an unknown reason, creating "metadata" directly in the object doesn't work
	// and the helper functions must be used
	j.SetKind("Job")
	j.SetAPIVersion("batch/v1")
	j.SetName(jobNameDataBackup)
	j.SetNamespace(namespaceEtcds)

	return j
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
