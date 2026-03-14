package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("[Admin API] Run Job action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	const runJobTestNamespace = "openshift-azure-operator"

	It("must stream job logs and report success for a completing job", func(ctx context.Context) {
		By("building a minimal Job manifest that echoes a known string")
		parallelism := int32(1)
		completions := int32(1)
		backoffLimit := int32(0)
		jobName := fmt.Sprintf("e2e-runjob-%d-%s", GinkgoParallelProcess(), utilrand.String(5))
		jobManifest := map[string]interface{}{
			"kind":       "Job",
			"apiVersion": "batch/v1",
			"metadata": map[string]interface{}{
				"name":      jobName,
				"namespace": runJobTestNamespace,
			},
			"spec": map[string]interface{}{
				"parallelism":  parallelism,
				"completions":  completions,
				"backoffLimit": backoffLimit,
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"restartPolicy": string(corev1.RestartPolicyNever),
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "worker",
								"image":   "image-registry.openshift-image-registry.svc:5000/openshift/cli:latest",
								"command": []interface{}{"/bin/bash", "-c", "echo hello-from-runjob"},
							},
						},
					},
				},
			},
		}

		By("posting the job manifest via the RP admin runjob API")
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/runjob", nil, false, jobManifest, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /runjob transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from runjob endpoint, got %d", resp.StatusCode)

		By("verifying the job output and lifecycle messages are present")
		Expect(output).To(ContainSubstring("hello-from-runjob"),
			"expected job log output in streamed response:\n%s", output)
		Expect(output).To(ContainSubstring("Job succeeded."),
			"expected 'Job succeeded.' in streamed response:\n%s", output)
		Expect(output).To(ContainSubstring("Cleanup complete."),
			"expected 'Cleanup complete.' in streamed response:\n%s", output)

		By("verifying the Job was deleted by the RP after completion")
		Eventually(func(g Gomega) {
			jobs, err := clients.Kubernetes.BatchV1().Jobs(runJobTestNamespace).List(ctx, metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred(), "listing jobs in %s", runJobTestNamespace)
			for _, j := range jobs.Items {
				g.Expect(j.Name).NotTo(ContainSubstring(jobName),
					"job should have been deleted by the RP cleanup")
			}
		}, "30s", "1s").Should(Succeed(), "RP cleanup did not delete the job within 30s")
	})

	It("must return 400 when request body is missing", func(ctx context.Context) {
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/runjob", nil, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /runjob transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for missing body, got %d", resp.StatusCode)
	})

	It("must return 400 when manifest kind is not Job", func(ctx context.Context) {
		reqBody := map[string]interface{}{"kind": "Pod"}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/runjob", nil, false, reqBody, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /runjob transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for non-Job kind, got %d", resp.StatusCode)
	})

	It("must return 403 when targeting a customer namespace", func(ctx context.Context) {
		reqBody := map[string]interface{}{
			"kind": "Job",
			"metadata": map[string]interface{}{
				"name":      "test-job",
				"namespace": "customer-app",
			},
		}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/runjob", nil, false, reqBody, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /runjob transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden),
			"expected 403 Forbidden for customer namespace, got %d", resp.StatusCode)
	})

	It("must stream failure message when the job container exits non-zero", func(ctx context.Context) {
		By("building a Job manifest whose container exits with a non-zero code")
		backoffLimit := int32(0)
		jobManifest := batchv1.Job{
			TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("e2e-runjob-fail-%d-%s", GinkgoParallelProcess(), utilrand.String(5)),
				Namespace: runJobTestNamespace,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit: &backoffLimit,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{{
							Name:    "worker",
							Image:   "image-registry.openshift-image-registry.svc:5000/openshift/cli:latest",
							Command: []string{"/bin/bash", "-c", "exit 1"},
						}},
					},
				},
			},
		}

		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/runjob", nil, false, jobManifest, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /runjob transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from runjob endpoint even on job failure, got %d", resp.StatusCode)

		By("verifying the failure is reported and cleanup still ran")
		Expect(output).To(ContainSubstring("Job failed."),
			"expected 'Job failed.' in streamed response:\n%s", output)
		Expect(output).To(ContainSubstring("Cleanup complete."),
			"expected 'Cleanup complete.' in streamed response:\n%s", output)
	})
})
