package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var (
	nameserverRegex                = regexp.MustCompile("nameserver [0-9.]*")
	resolvConfJobIsCompleteTimeout = 2 * time.Minute
	resolvConfJobIsCompletePolling = 5 * time.Second
	verifyResolvConfTimeout        = 30 * time.Second
	verifyResolvConfPollInterval   = 1 * time.Second
	nodesReadyPollInterval         = 2 * time.Second
	nicUpdateWaitTime              = 5 * time.Second
)

const (
	maxObjNameLen           = 63
	resolvConfContainerName = "read-resolv-conf"
)

var _ = Describe("ARO cluster DNS", Label(regressiontest), func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must not be adversely affected by Azure host servicing", func(ctx context.Context) {
		By("creating a test namespace")
		testNamespace := fmt.Sprintf("test-e2e-%d", GinkgoParallelProcess())
		p := BuildNewProject(ctx, clients.Kubernetes, clients.Project, testNamespace)

		By("verifying the namespace is ready")
		Eventually(func(ctx context.Context) error {
			return p.VerifyProjectIsReady(ctx)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		DeferCleanup(func(ctx context.Context) {
			By("deleting the test namespace")
			p.CleanUp(ctx)
		})

		By("listing all cluster nodes to retrieve the names of the worker nodes")
		workerNodes := map[string]string{}
		nodeList := ListK8sObjectWithRetry(
			ctx, clients.Kubernetes.CoreV1().Nodes().List, metav1.ListOptions{},
		)

		for _, node := range nodeList.Items {
			name := node.Name
			if isWorkerNode(node) {
				workerNodes[name] = ""
			}
		}

		By("getting each worker node's private IP address")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		clusterResourceGroup := stringutils.LastTokenByte(*oc.ClusterProfile.ResourceGroupID, '/')
		for wn := range workerNodes {
			resp, err := clients.Interfaces.Get(ctx, clusterResourceGroup, nicName(wn), nil)
			Expect(err).NotTo(HaveOccurred())
			nic := resp.Interface

			Expect(nic.Properties).NotTo(BeNil())
			Expect(nic.Properties.IPConfigurations).To(HaveLen(1))
			Expect(nic.Properties.IPConfigurations[0].Properties).NotTo(BeNil())
			Expect(nic.Properties.IPConfigurations[0].Properties.PrivateIPAddress).NotTo(BeNil())
			workerNodes[wn] = *nic.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		}

		By("preparing to read resolv.conf from each of the worker nodes by allowing the test namespace's ServiceAccount to use the hostmount-anyuid SecurityContextConstraint")
		sccName := fmt.Sprintf("system:serviceaccount:%s:default", testNamespace)

		// This is wrapped in an Eventually call with some retries to avoid test flakes on the off chance that
		// OCP happens to be trying to do stuff to the SecurityContextConstraint at the same time as us.
		Eventually(func() error {
			scc, err := clients.SecurityClient.SecurityV1().SecurityContextConstraints().Get(ctx, "hostmount-anyuid", metav1.GetOptions{})
			if err != nil {
				return err
			}

			if scc.Users == nil {
				scc.Users = []string{}
			}
			scc.Users = append(scc.Users, sccName)
			_, err = clients.SecurityClient.SecurityV1().SecurityContextConstraints().Update(ctx, scc, metav1.UpdateOptions{})
			return err
		}).WithContext(ctx).
			WithTimeout(10 * time.Second).
			WithPolling(1 * time.Second).
			Should(Succeed())

		DeferCleanup(func(ctx context.Context) {
			By("removing the test namespace's ServiceAccount's ability to use the hostmount-anyuid SecurityContextConstraint")
			Eventually(func() error {
				scc, err := clients.SecurityClient.SecurityV1().SecurityContextConstraints().Get(ctx, "hostmount-anyuid", metav1.GetOptions{})
				if err != nil {
					return err
				}

				if scc.Users == nil {
					scc.Users = []string{}
				}
				idx := -1
				for i, u := range scc.Users {
					if u == sccName {
						idx = i
						break
					}
				}
				if idx >= 0 {
					users := []string{}
					for i, u := range scc.Users {
						if i == idx {
							continue
						}
						users = append(users, u)
					}
					scc.Users = users
					_, err = clients.SecurityClient.SecurityV1().SecurityContextConstraints().Update(ctx, scc, metav1.UpdateOptions{})
					return err
				}

				return nil
			}).WithContext(ctx).
				WithTimeout(10 * time.Second).
				WithPolling(1 * time.Second).
				Should(Succeed())
		})

		By("verifying each worker node's resolv.conf via a one-shot Job per node")
		for wn, ip := range workerNodes {
			createResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
			resolvConfJobIsComplete(ctx, clients.Kubernetes, wn, testNamespace)

			Eventually(verifyResolvConf).
				WithContext(ctx).
				WithTimeout(verifyResolvConfTimeout).
				WithPolling(verifyResolvConfPollInterval).
				WithArguments(clients.Kubernetes, wn, testNamespace, ip).
				Should(Succeed())

			deleteResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
		}

		By("stopping all three worker VMs")
		for wn := range workerNodes {
			err = clients.VirtualMachines.StopAndWait(ctx, clusterResourceGroup, wn, false)
			Expect(err).NotTo(HaveOccurred())
		}

		By("disabling accelerated networking on all three worker VMs to simulate host servicing")
		for wn := range workerNodes {
			err = toggleAcceleratedNetworking(ctx, clients.Interfaces, clusterResourceGroup, wn, false)
			Expect(err).NotTo(HaveOccurred())
		}

		// A small buffer here will help us be confident that the NIC API call to toggle accelerated
		// networking will have taken full effect before we move forward under the assumption
		// that it has.
		time.Sleep(nicUpdateWaitTime)

		By("restarting the three worker VMs")
		for wn := range workerNodes {
			err = clients.VirtualMachines.StartAndWait(ctx, clusterResourceGroup, wn)
			Expect(err).NotTo(HaveOccurred())
		}

		By("waiting for all nodes to return to a Ready state")
		Eventually(workerNodesReady).
			WithContext(ctx).
			WithTimeout(DefaultEventuallyTimeout).
			WithPolling(nodesReadyPollInterval).
			WithArguments(clients.Kubernetes).
			Should(Succeed())

		By("verifying each worker node's resolv.conf is still correct after simulating host servicing, again via a one-shot Job per node")

		for wn, ip := range workerNodes {
			createResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
			resolvConfJobIsComplete(ctx, clients.Kubernetes, wn, testNamespace)

			Eventually(verifyResolvConf).
				WithContext(ctx).
				WithTimeout(verifyResolvConfTimeout).
				WithPolling(verifyResolvConfPollInterval).
				WithArguments(clients.Kubernetes, wn, testNamespace, ip).
				Should(Succeed())

			deleteResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
		}

		By("stopping all three worker VMs")
		for wn := range workerNodes {
			err = clients.VirtualMachines.StopAndWait(ctx, clusterResourceGroup, wn, false)
			Expect(err).NotTo(HaveOccurred())
		}

		By("re-enabling accelerated networking on all three worker VMs")
		for wn := range workerNodes {
			err = toggleAcceleratedNetworking(ctx, clients.Interfaces, clusterResourceGroup, wn, true)
			Expect(err).NotTo(HaveOccurred())
		}

		time.Sleep(nicUpdateWaitTime)

		By("restarting the three worker VMs")
		for wn := range workerNodes {
			err = clients.VirtualMachines.StartAndWait(ctx, clusterResourceGroup, wn)
			Expect(err).NotTo(HaveOccurred())
		}

		By("waiting for all nodes to return to a Ready state")
		Eventually(workerNodesReady).
			WithContext(ctx).
			WithTimeout(DefaultEventuallyTimeout).
			WithPolling(nodesReadyPollInterval).
			WithArguments(clients.Kubernetes).
			Should(Succeed())
	})
})

func nicName(nodeName string) string {
	return fmt.Sprintf("%s-nic", nodeName)
}

func resolvConfJobName(nodeName string) string {
	jobName := fmt.Sprintf("read-resolv-conf-%s", nodeName)
	if len(jobName) > maxObjNameLen {
		jobName = jobName[:maxObjNameLen]
	}
	return jobName
}

func createResolvConfJob(ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string) {
	hpt := corev1.HostPathFile
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: resolvConfJobName(nodeName),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName: nodeName,
					Containers: []corev1.Container{
						{
							Name:  resolvConfContainerName,
							Image: "busybox",
							Command: []string{
								"/bin/sh",
								"-c",
								"cat /tmp/resolv.conf",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "resolv-conf",
									MountPath: "/tmp/resolv.conf",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "resolv-conf",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/resolv.conf",
									Type: &hpt,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
	CreateK8sObjectWithRetry(ctx, cli.BatchV1().Jobs(namespace).Create, job, metav1.CreateOptions{})
}

func resolvConfJobIsComplete(ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string) {
	Eventually(func(ctx context.Context) (bool, error) {
		job, err := cli.BatchV1().Jobs(namespace).Get(ctx, resolvConfJobName(nodeName), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return job.Status.Succeeded == 1, nil
	}).
		WithContext(ctx).
		WithTimeout(resolvConfJobIsCompleteTimeout).
		WithPolling(resolvConfJobIsCompletePolling).
		Should(BeTrue())
}

func deleteResolvConfJob(ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string) {
	dpb := metav1.DeletePropagationBackground
	deleteFunc := cli.BatchV1().Jobs(namespace).Delete
	DeleteK8sObjectWithRetry(ctx, deleteFunc, resolvConfJobName(nodeName), metav1.DeleteOptions{
		PropagationPolicy: &dpb,
	})
}

func verifyResolvConf(
	ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string, nodeIp string,
) error {
	jobName := resolvConfJobName(nodeName)
	podList := ListK8sObjectWithRetry(ctx, cli.CoreV1().Pods(namespace).List, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if len(podList.Items) != 1 {
		return fmt.Errorf("found %v Pods associated with the Job, but there should be exactly 1", len(podList.Items))
	}

	podName := podList.Items[0].Name
	tailLines := int64(10)
	podLogOptions := &corev1.PodLogOptions{
		Container: resolvConfContainerName,
		Follow:    false,
		TailLines: &tailLines,
	}
	podLogRequest := cli.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions)
	stream, err := podLogRequest.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	logs := []string{}
	for {
		buf := make([]byte, 2000)
		numBytes, err := stream.Read(buf)
		if numBytes == 0 {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		logs = append(logs, string(buf[:numBytes]))
	}

	log := strings.Join(logs, "\n")
	if log == "" {
		return fmt.Errorf("pod didn't have any logs")
	}

	nameserverLine := nameserverRegex.FindString(log)
	if nameserverLine == "" {
		return fmt.Errorf("didn't find nameserver in resolv.conf")
	}

	_, nameserverIp, _ := strings.Cut(nameserverLine, " ")
	if nameserverIp != nodeIp {
		return fmt.Errorf("nameserver specified in resolv.conf does not match node's IP address")
	}

	return nil
}

func toggleAcceleratedNetworking(ctx context.Context, interfaces armnetwork.InterfacesClient, clusterResourceGroup string, nodeName string, enabled bool) error {
	resp, err := interfaces.Get(ctx, clusterResourceGroup, nicName(nodeName), nil)
	if err != nil {
		return err
	}
	nic := resp.Interface

	if nic.Properties == nil {
		return fmt.Errorf("NIC properties are nil")
	}
	nic.Properties.EnableAcceleratedNetworking = to.BoolPtr(enabled)
	err = clients.Interfaces.CreateOrUpdateAndWait(ctx, clusterResourceGroup, nicName(nodeName), nic, nil)
	return err
}

func isWorkerNode(node corev1.Node) bool {
	ok := false
	if node.Labels != nil {
		_, ok = node.Labels["node-role.kubernetes.io/worker"]
	}
	return ok
}

func workerNodesReady(ctx context.Context, cli kubernetes.Interface) error {
	nodeList, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodeList.Items {
		if isWorkerNode(node) && !ready.NodeIsReady(&node) {
			return fmt.Errorf("a worker node is not yet ready")
		}
	}

	return nil
}
