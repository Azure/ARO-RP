package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Image Registry pull-secret reconciler
// Users tend to do damage to corev1.Secret openshift-config/pull-secret
// this controllers ensures valid ARO secret for Azure mirror with
// openshift images
// It also signals presense of Red Hat image registry keys via operator conditions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}
var rhKeys = []string{"registry.redhat.io", "cloud.redhat.com"}

// PullSecretReconciler reconciles a Cluster object
type PullSecretReconciler struct {
	kubernetescli kubernetes.Interface
	arocli        aroclient.Interface
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.Interface) *PullSecretReconciler {
	return &PullSecretReconciler{
		log:           log,
		kubernetescli: kubernetescli,
		arocli:        arocli,
	}
}

// Reconcile will make sure that the ACR part of the pull secret is correct. The
// conditions under which Reconcile is called are slightly unusual and are as
// follows:
// * If the Cluster object changes, we'll see the *Cluster* object requested.
// * If a Secret object owned by the Cluster object changes (e.g., but not
//   limited to, the configuration Secret, we'll see the *Cluster* object
//   requested).
// * If the pull Secret object (which is not owned by the Cluster object)
//   changes, we'll see the pull Secret object requested.
func (r *PullSecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()

	operatorSecret, err := r.kubernetescli.CoreV1().Secrets(operator.Namespace).Get(ctx, operator.SecretName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	var userSecret *corev1.Secret

	userSecret, err = r.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Get(ctx, pullSecretName.Name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	// reconcile global pull secret
	// detects if the global pull secret is broken and fixes it by using backup managed by ARO operator
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fix pull secret if its broken to have at least the ARO pull secret
		return r.fixAndUpdateGlobalPullSecret(ctx, operatorSecret, userSecret)
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	redHatKeyCondition := r.buildRedHatKeyCondition(userSecret)
	err = controllers.SetCondition(ctx, r.arocli, redHatKeyCondition, operator.RoleMaster)

	return reconcile.Result{}, err
}

// SetupWithManager setup our manager
func (r *PullSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pullSecretPredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return (meta.GetName() == pullSecretName.Name && meta.GetNamespace() == pullSecretName.Namespace) ||
			(meta.GetName() == operator.SecretName && meta.GetNamespace() == operator.Namespace)
	})

	aroClusterPredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return meta.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to For(&v1.Secret{})., but can't call For multiple times on one builder
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(pullSecretPredicate),
		).
		Named(controllers.PullSecretControllerName).
		Complete(r)
}

// fixAndUpdateGlobalPullSecret checks the state of the pull secrets, in case of missing or broken ARO pull secret
// it replaces it with working one from controller Secret
func (r *PullSecretReconciler) fixAndUpdateGlobalPullSecret(ctx context.Context, operatorSecret, userSecret *corev1.Secret) (err error) {
	if operatorSecret == nil {
		return errors.New("nil operator secret, cannot verify userData integrity")
	}

	var secret *corev1.Secret
	create := userSecret == nil
	remove := false

	// userSecret can happen to have broken type, which means it have to be recreated
	// with proper type
	if userSecret != nil && (userSecret.Type != corev1.SecretTypeDockerConfigJson || userSecret.Data == nil) {
		create = true
		remove = true
	}

	if create {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		secret.Data = make(map[string][]byte)
	} else {
		secret = userSecret.DeepCopy()
		if !json.Valid(secret.Data[corev1.DockerConfigJsonKey]) {
			delete(secret.Data, corev1.DockerConfigJsonKey)
		}
	}

	fixedData, update, err := pullsecret.Merge(string(secret.Data[corev1.DockerConfigJsonKey]), string(operatorSecret.Data[corev1.DockerConfigJsonKey]))
	if err != nil {
		return err
	}

	if !create && !update {
		return nil
	}

	if remove {
		// unfortunately the type field is immutable.
		err := r.kubernetescli.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}
	}

	secret.Data[corev1.DockerConfigJsonKey] = []byte(fixedData)

	if create {
		_, err := r.kubernetescli.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})

		return err
	}

	_, err = r.kubernetescli.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})

	return err
}

func (r *PullSecretReconciler) buildRedHatKeyCondition(secret *corev1.Secret) *status.Condition {
	// parse keys and validate JSON
	parsedKeys, err := pullsecret.UnmarshalSecretData(secret)
	foundKeys := []string{}
	failed := false
	if err != nil {
		r.log.Info("pull secret is not valid json - recreating")
		failed = true
	} else {
		foundKeys = r.checkRHRegistryKey(parsedKeys)
	}

	keyCondition := r.keyCondition(failed, foundKeys)
	return keyCondition
}

// checkRHRegistryKey checks whether the rhRegistry key:
//   - redhat.registry.io
//   - cloud.redhat.com
// is present in the pullSecret
func (r *PullSecretReconciler) checkRHRegistryKey(psData map[string]string) (foundKeys []string) {
	if psData == nil {
		return foundKeys
	}

	for _, rhKey := range rhKeys {
		for k, v := range psData {
			if k == rhKey && len(v) > 0 {
				foundKeys = append(foundKeys, rhKey)
			}
		}

	}

	return foundKeys
}

func (r *PullSecretReconciler) keyCondition(failed bool, foundKeys []string) *status.Condition {
	keyCondition := &status.Condition{
		Type:   arov1alpha1.RedHatKeyPresent,
		Status: corev1.ConditionFalse,
		Reason: "CheckFailed",
	}

	if failed {
		keyCondition.Message = "Cannot parse pull-secret"
		return keyCondition
	}

	keyCondition.Reason = "CheckDone"

	if len(foundKeys) == 0 {
		keyCondition.Message = "No Red Hat key found in pull-secret"
		return keyCondition
	}

	if len(foundKeys) == 0 {
		keyCondition.Status = corev1.ConditionFalse
		return keyCondition
	}

	b := strings.Builder{}
	for _, key := range foundKeys {
		b.WriteString(key)
		b.WriteString(",")
	}

	keyCondition.Status = corev1.ConditionTrue
	keyCondition.Message = b.String()

	return keyCondition
}
