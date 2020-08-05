package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

func updatedObjects() ([]string, error) {
	pods, err := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").List(metav1.ListOptions{
		LabelSelector: "app=aro-operator-master",
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("%d aro-operator-master pods found", len(pods.Items))
	}
	b, err := clients.Kubernetes.CoreV1().Pods("openshift-azure-operator").GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw()
	if err != nil {
		return nil, err
	}

	var result []string
	rx := regexp.MustCompile(`.*msg="(Update|Create) ([a-zA-Z\/.]+).*`)
	changes := rx.FindAllStringSubmatch(string(b), -1)
	if len(changes) > 0 {
		for _, change := range changes {
			if len(change) == 3 {
				result = append(result, change[1]+" "+change[2])
			}
		}
	} else {
		log.Warnf("FindAllStringSubmatch: returned %v", changes)
	}
	return result, nil
}

var _ = Describe("ARO Operator - Internet checking", func() {
	var originalURLs []string
	BeforeEach(func() {
		// save the originalURLs
		co, err := clients.AROClusters.Clusters().Get("cluster", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			Skip("skipping tests as aro-operator is not deployed")
		}

		Expect(err).NotTo(HaveOccurred())
		originalURLs = co.Spec.InternetChecker.URLs
	})
	AfterEach(func() {
		// set the URLs back again
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.Clusters().Get("cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = originalURLs
			_, err = clients.AROClusters.Clusters().Update(co)
			return err
		})
		Expect(err).NotTo(HaveOccurred())
	})
	Specify("the InternetReachable default list should all be reachable", func() {
		co, err := clients.AROClusters.Clusters().Get("cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(co.Status.Conditions.IsTrueFor(arov1alpha1.InternetReachableFromMaster)).To(BeTrue())
		Expect(co.Status.Conditions.IsTrueFor(arov1alpha1.InternetReachableFromWorker)).To(BeTrue())
	})
	Specify("custom invalid site shows not InternetReachable", func() {
		// set an unreachable URL
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.Clusters().Get("cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.URLs = []string{"https://localhost:1234/shouldnotexist"}
			_, err = clients.AROClusters.Clusters().Update(co)
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		// confirm the conditions are correct
		err = wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) {
			co, err := clients.AROClusters.Clusters().Get("cluster", metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			log.Info(co.Status.Conditions)
			return co.Status.Conditions.IsFalseFor(arov1alpha1.InternetReachableFromMaster) &&
				co.Status.Conditions.IsFalseFor(arov1alpha1.InternetReachableFromWorker), nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("ARO Operator - Geneva Logging", func() {
	BeforeEach(func() {
		_, err := clients.AROClusters.Clusters().Get("cluster", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			Skip("skipping tests as aro-operator is not deployed")
		}
	})
	Specify("genevalogging must be repaired if deployment deleted", func() {
		mdsdReady := ready.CheckDaemonSetIsReady(clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging"), "mdsd")

		err := wait.PollImmediate(30*time.Second, 15*time.Minute, mdsdReady)
		Expect(err).NotTo(HaveOccurred())
		initial, err := updatedObjects()
		Expect(err).NotTo(HaveOccurred())

		// delete the mdsd daemonset
		err = clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete("mdsd", nil)
		Expect(err).NotTo(HaveOccurred())

		// Wait for it to be fixed
		err = wait.PollImmediate(30*time.Second, 15*time.Minute, mdsdReady)
		Expect(err).NotTo(HaveOccurred())

		// confirm that only one object was updated
		final, err := updatedObjects()
		Expect(err).NotTo(HaveOccurred())
		if len(final)-len(initial) != 1 {
			log.Error("initial changes ", initial)
			log.Error("final changes ", final)
		}
		Expect(len(final) - len(initial)).To(Equal(1))
	})
})
