package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	OVSFlowContianerName string = "container-00"
	OVSFlowPodName       string = "aro-ovsflows"
	OVSFlowPodNamespace  string = "default"
)

func (f *frontend) getOVSFlows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	config, err := f.getKubernetesClientSet(ctx, r, log)
	if err != nil {
		log.Error(err)
	}

	client := Client{config}
	// Main logic written in f._getOSVFlow() func
	b, err := client._getOVSFlows(ctx, r, log, config)
	if err != nil {
		log.Error(err)
	}
	reply(log, w, nil, b, err)
}

func (client Client) _getOVSFlows(ctx context.Context, r *http.Request, log *logrus.Entry, config *kubernetes.Clientset) ([]byte, error) {
	//Checking if the node mentioned in the query is correct or not.
	nodeName := r.URL.Query().Get("node")
	if nodeName == "" {
		err := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "Invalid Node Name")
		return nil, err
	}

	// Checking if the given node exists
	node, err := config.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Creating a pod which will run as privileged in the given node to execute "$ ovs-ofctl -O OpenFlow13 dump-flows br0" on that node to extract the OVS flows.
	pod, err := client.createOVSFlowPod(ctx, node.Name, log)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	//  The below loop waits of the phase of the created pod to be Succeeded, if not it times out after 60 seconds and returns error.
	for i := 0; pod.Status.Phase != corev1.PodSucceeded; i++ {
		if i == 60 {
			err = api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeInternalServerError, "", "")
			return nil, err
		}
		pod, err = config.CoreV1().Pods(OVSFlowPodNamespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			log.Error(err)
			return nil, err
		}
		time.Sleep(time.Second)
	}

	// The OVS flows would be present in the logs of the created pod.
	b, err := client.getPodLogs(ctx, log, pod)
	if err != nil {
		return nil, err
	}

	// Clearing the pod after extracting its logs.
	defer func() {
		config.CoreV1().Pods(OVSFlowPodNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Error(err)
		}
	}()

	return b, nil
}

type Client struct {
	Clientset kubernetes.Interface
}

// This Function extract logs from the passed pod and return in the form of a slice bytes.
func (c Client) getPodLogs(ctx context.Context, log *logrus.Entry, pod *corev1.Pod) ([]byte, error) {
	opts := corev1.PodLogOptions{Container: OVSFlowContianerName}
	r := c.Clientset.CoreV1().Pods(OVSFlowPodNamespace).GetLogs(pod.Name, &opts)
	result, err := r.Do(ctx).Raw()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return result, err
}

func (c Client) createOVSFlowPod(ctx context.Context, node string, log *logrus.Entry) (*corev1.Pod, error) {
	var hostpathType corev1.HostPathType = "Directory"
	hostPath := corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{"/", &hostpathType}}
	volumes := []corev1.Volume{{"host", hostPath}}
	privileged := true
	var runAsUser int64 = 0
	i := 0
	podName := OVSFlowPodName
	for {
		pod := &corev1.Pod{
			metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			metav1.ObjectMeta{
				Name:      podName,
				Namespace: OVSFlowPodNamespace,
			},
			corev1.PodSpec{
				Volumes: volumes,
				Containers: []corev1.Container{{
					Name:    OVSFlowContianerName,
					Image:   "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:103505c93bf45c4a29301f282f1ff046e35b63bceaf4df1cca2e631039289da2",
					Command: []string{"chroot", "/host", "/bin/bash", "-c", "ovs-ofctl -O OpenFlow13 dump-flows br0"},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "host",
						MountPath: "/host",
					}},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
						RunAsUser:  &runAsUser,
					},
				}},
				RestartPolicy: "Never",
				NodeName:      node,
				HostNetwork:   true,
				HostPID:       true,
			},
			corev1.PodStatus{},
		}
		pod, err := c.Clientset.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
		if err != nil && err.Error() == fmt.Sprintf("pods \"%s\" already exists", podName) {
			i++
			podName = fmt.Sprintf("%s-%d", OVSFlowPodName, i)
			continue
		} else if err != nil {
			log.Error(err)
			return nil, err
		}
		return pod, nil
	}
}

func (f *frontend) getKubernetesClientSet(ctx context.Context, r *http.Request, log *logrus.Entry) (*kubernetes.Clientset, error) {
	vars := mux.Vars(r)
	r.URL.Path = filepath.Dir(r.URL.Path)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	// Getting OpenShiftCluster Document
	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		log.Error(api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]))
		return nil, err
	case err != nil:
		return nil, err
	}

	// Getting rest config to create *kubernetes.Clientset
	restConfig, err := restconfig.RestConfig(f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}
	config, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
