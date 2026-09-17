package aksjob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/installer/inputs"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEnsureInstallerResources(t *testing.T) {
	ctx := context.Background()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: Namespace},
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: Namespace,
			Annotations: map[string]string{
				"azure.workload.identity/client-id": "identity-client-id",
			},
		},
	}
	client := fake.NewSimpleClientset(namespace, serviceAccount)
	_, log := testlog.New()
	manager := newManager(log, client)
	installerInputs := &inputs.InstallerInputs{
		ClusterJSON:               []byte(`{"cluster":"document"}`),
		SubscriptionJSON:          []byte(`{"subscription":"document"}`),
		ServicePrincipalJSON:      []byte(`{"clientId":"client-id"}`),
		PullSecretJSON:            []byte(`{"auths":{}}`),
		CustomManifests:           map[string][]byte{"manifest.yaml": []byte("apiVersion: v1\n")},
		EnvironmentVariables:      map[string]string{"ARO_UUID": "cluster-uuid"},
		ClusterUUID:               "cluster-uuid",
		SubscriptionID:            "subscription-id",
		Namespace:                 Namespace,
		ExecutionID:               "execution-id",
		JobName:                   JobName("execution-id"),
		InstallerPullspec:         "example.invalid/installer@sha256:1234",
		InstallerIdentityClientID: "identity-client-id",
		Timeout:                   time.Hour,
	}

	if err := manager.ensureNamespace(ctx, installerInputs); err != nil {
		t.Fatal(err)
	}
	if err := manager.ensureServiceAccount(ctx, installerInputs); err != nil {
		t.Fatal(err)
	}
	if err := manager.ensureSecrets(ctx, installerInputs); err != nil {
		t.Fatal(err)
	}
	if err := manager.ensureJob(ctx, installerInputs); err != nil {
		t.Fatal(err)
	}

	gotServiceAccount, err := client.CoreV1().ServiceAccounts(installerInputs.Namespace).Get(ctx, ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := gotServiceAccount.Annotations["azure.workload.identity/client-id"]; got != "identity-client-id" {
		t.Errorf("workload identity client ID = %q, want identity-client-id", got)
	}

	job, err := client.BatchV1().Jobs(installerInputs.Namespace).Get(ctx, installerInputs.JobName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if job.Spec.BackoffLimit == nil || *job.Spec.BackoffLimit != 0 {
		t.Errorf("backoffLimit = %v, want 0", job.Spec.BackoffLimit)
	}
	podSpec := job.Spec.Template.Spec
	if podSpec.ServiceAccountName != ServiceAccountName {
		t.Errorf("serviceAccountName = %q, want %q", podSpec.ServiceAccountName, ServiceAccountName)
	}
	if len(podSpec.ImagePullSecrets) != 1 || podSpec.ImagePullSecrets[0].Name != installerInputs.JobName+"-pull" {
		t.Errorf("imagePullSecrets = %#v", podSpec.ImagePullSecrets)
	}
	container := podSpec.Containers[0]
	if container.WorkingDir != "/.azure" {
		t.Errorf("workingDir = %q, want /.azure", container.WorkingDir)
	}

	wantMounts := map[string]bool{
		"/.azure":                         false,
		"/.azure/99_aro.json":             false,
		"/.azure/99_sub.json":             false,
		"/.azure/osServicePrincipal.json": false,
		"/manifests":                      false,
		"/output":                         false,
		"/tmp":                            false,
	}
	for _, mount := range container.VolumeMounts {
		if _, ok := wantMounts[mount.MountPath]; ok {
			wantMounts[mount.MountPath] = true
		}
	}
	for path, found := range wantMounts {
		if !found {
			t.Errorf("missing volume mount %s", path)
		}
	}
}

func TestJobCompletedReturnsFailure(t *testing.T) {
	ctx := context.Background()
	namespace := "aro-install-cluster-uuid"
	jobName := JobName("execution-id")
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: namespace},
		Status: batchv1.JobStatus{
			Failed: 1,
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installer-pod",
			Namespace: namespace,
			Labels:    map[string]string{"job-name": jobName},
		},
	}
	client := fake.NewSimpleClientset(job, pod)
	_, log := testlog.New()
	manager := newManager(log, client)

	done, err := manager.jobCompleted(ctx, &inputs.InstallerInputs{Namespace: namespace, JobName: jobName})
	if !done {
		t.Fatal("expected completed Job")
	}
	if err == nil {
		t.Fatal("expected failed Job to return an error")
	}
}

func TestEnsureJobReattachesToExecution(t *testing.T) {
	ctx := context.Background()
	namespace := "aro-install-cluster-uuid"
	jobName := JobName("execution-id")
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"execution-id":     "execution-id",
				"aro-cluster-uuid": "cluster-uuid",
			},
		},
	}
	client := fake.NewSimpleClientset(job)
	_, log := testlog.New()
	manager := newManager(log, client)

	err := manager.ensureJob(ctx, &inputs.InstallerInputs{
		Namespace:   namespace,
		ExecutionID: "execution-id",
		ClusterUUID: "cluster-uuid",
		JobName:     jobName,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanupRetainsSharedAndOtherExecutionResources(t *testing.T) {
	ctx := context.Background()
	jobName := JobName("execution-id")
	otherJobName := JobName("other-execution-id")
	names := resourceNames(jobName)
	otherNames := resourceNames(otherJobName)
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: Namespace}}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: ServiceAccountName, Namespace: Namespace},
	}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: Namespace}}
	otherJob := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: otherJobName, Namespace: Namespace}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: names.inputsSecret, Namespace: Namespace}}
	otherSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: otherNames.inputsSecret, Namespace: Namespace}}
	client := fake.NewSimpleClientset(namespace, serviceAccount, job, otherJob, secret, otherSecret)
	_, log := testlog.New()
	manager := newManager(log, client)

	if err := manager.Cleanup(ctx, Namespace, jobName); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().Namespaces().Get(ctx, Namespace, metav1.GetOptions{}); err != nil {
		t.Fatalf("shared namespace was removed: %v", err)
	}
	if _, err := client.CoreV1().ServiceAccounts(Namespace).Get(ctx, ServiceAccountName, metav1.GetOptions{}); err != nil {
		t.Fatalf("shared ServiceAccount was removed: %v", err)
	}
	if _, err := client.BatchV1().Jobs(Namespace).Get(ctx, jobName, metav1.GetOptions{}); err == nil {
		t.Fatal("execution Job was not removed")
	}
	if _, err := client.CoreV1().Secrets(Namespace).Get(ctx, names.inputsSecret, metav1.GetOptions{}); err == nil {
		t.Fatal("execution Secret was not removed")
	}
	if _, err := client.BatchV1().Jobs(Namespace).Get(ctx, otherJobName, metav1.GetOptions{}); err != nil {
		t.Fatalf("other execution Job was removed: %v", err)
	}
	if _, err := client.CoreV1().Secrets(Namespace).Get(ctx, otherNames.inputsSecret, metav1.GetOptions{}); err != nil {
		t.Fatalf("other execution Secret was removed: %v", err)
	}
}

func TestTruncateInstallerLogs(t *testing.T) {
	logs := string(make([]byte, maxInstallerLogBytes+1))
	got := truncateInstallerLogs(logs)
	if len(got) >= len(logs) {
		t.Fatalf("truncated log length = %d, original = %d", len(got), len(logs))
	}
}
