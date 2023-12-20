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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/test/util/project"
)

var (
	nameserverRegex              = regexp.MustCompile("nameserver [0-9.]*")
	verifyResolvConfTimeout      = 30 * time.Second
	verifyResolvConfPollInterval = 1 * time.Second
	nodesReadyPollInterval       = 2 * time.Second
	nicUpdateWaitTime            = 5 * time.Second
)

const (
	maxObjNameLen           = 63
	resolvConfContainerName = "read-resolv-conf"
)

var _ = Describe("ARO cluster DNS", func() {
	It("must not be adversely affected by Azure host servicing", func(ctx context.Context) {
		By("creating a test namespace")
		testNamespace := fmt.Sprintf("test-e2e-%d", GinkgoParallelProcess())
		p := project.NewProject(clients.Kubernetes, clients.Project, testNamespace)
		err := p.Create(ctx)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")

		By("verifying the namespace is ready")
		Eventually(func(ctx context.Context) error {
			return p.Verify(ctx)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeNil())

		DeferCleanup(func(ctx context.Context) {
			By("deleting the test namespace")
			err := p.Delete(ctx)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete test namespace")

			By("verifying the namespace is deleted")
			Eventually(func(ctx context.Context) error {
				return p.VerifyProjectIsDeleted(ctx)
			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(BeNil())
		})

		By("listing all cluster nodes to retrieve the names of the worker nodes")
		workerNodes := map[string]string{}
		nodeList, err := clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		for _, node := range nodeList.Items {
			name := node.ObjectMeta.Name
			if strings.Contains(name, "worker") {
				workerNodes[name] = ""
			}
		}
		Expect(workerNodes).To(HaveLen(3))

		By("getting each worker node's private IP address")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())

		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')
		for wn := range workerNodes {
			nic, err := clients.Interfaces.Get(ctx, clusterResourceGroup, nicName(wn), "")
			Expect(err).NotTo(HaveOccurred())

			Expect(nic.InterfacePropertiesFormat).NotTo(BeNil())
			Expect(nic.IPConfigurations).NotTo(BeNil())
			Expect(*nic.IPConfigurations).To(HaveLen(1))
			Expect((*nic.IPConfigurations)[0].InterfaceIPConfigurationPropertiesFormat).NotTo(BeNil())
			Expect((*nic.IPConfigurations)[0].PrivateIPAddress).NotTo(BeNil())
			workerNodes[wn] = *(*nic.IPConfigurations)[0].PrivateIPAddress
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
			err = createResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
			Expect(err).NotTo(HaveOccurred())

			Eventually(verifyResolvConf).
				WithContext(ctx).
				WithTimeout(verifyResolvConfTimeout).
				WithPolling(verifyResolvConfPollInterval).
				WithArguments(clients.Kubernetes, wn, testNamespace, ip).
				Should(Succeed())

			err = deleteResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
			Expect(err).NotTo(HaveOccurred())
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
		Eventually(nodesReady).
			WithContext(ctx).
			WithTimeout(DefaultEventuallyTimeout).
			WithPolling(nodesReadyPollInterval).
			WithArguments(clients.Kubernetes).
			Should(Succeed())

		By("verifying each worker node's resolv.conf is still correct after simulating host servicing, again via a one-shot Job per node")

		for wn, ip := range workerNodes {
			err = createResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
			Expect(err).NotTo(HaveOccurred())

			Eventually(verifyResolvConf).
				WithContext(ctx).
				WithTimeout(verifyResolvConfTimeout).
				WithPolling(verifyResolvConfPollInterval).
				WithArguments(clients.Kubernetes, wn, testNamespace, ip).
				Should(Succeed())

			err = deleteResolvConfJob(ctx, clients.Kubernetes, wn, testNamespace)
			Expect(err).NotTo(HaveOccurred())
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
		Eventually(nodesReady).
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

func createResolvConfJob(ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string) error {
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
	_, err := cli.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	return err
}

func deleteResolvConfJob(ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string) error {
	dpb := metav1.DeletePropagationBackground
	return cli.BatchV1().Jobs(namespace).Delete(ctx, resolvConfJobName(nodeName), metav1.DeleteOptions{
		PropagationPolicy: &dpb,
	})
}

func verifyResolvConf(ctx context.Context, cli kubernetes.Interface, nodeName string, namespace string, nodeIp string) error {
	jobName := resolvConfJobName(nodeName)
	podList, err := cli.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		return fmt.Errorf("found %v Pods associated with the Job, but there should be exactly 1", len(podList.Items))
	}

	podName := podList.Items[0].ObjectMeta.Name
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

func toggleAcceleratedNetworking(ctx context.Context, interfaces network.InterfacesClient, clusterResourceGroup string, nodeName string, enabled bool) error {
	nic, err := interfaces.Get(ctx, clusterResourceGroup, nicName(nodeName), "")
	if err != nil {
		return err
	}

	nic.EnableAcceleratedNetworking = to.BoolPtr(enabled)
	err = clients.Interfaces.CreateOrUpdateAndWait(ctx, clusterResourceGroup, nicName(nodeName), nic)
	return err
}

func nodesReady(ctx context.Context, cli kubernetes.Interface) error {
	nodeList, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodeList.Items {
		var readyCondition corev1.NodeCondition
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				readyCondition = condition
			}
		}
		if (readyCondition == corev1.NodeCondition{}) {
			return fmt.Errorf("unable to check if a node is ready")
		}
		if readyCondition.Status != corev1.ConditionTrue {
			return fmt.Errorf("a node is not yet ready")
		}
	}

	return nil
}
