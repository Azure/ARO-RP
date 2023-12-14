package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Image Registry pull-secret reconciler
// Users tend to do damage to corev1.Secret openshift-config/pull-secret
// this controllers ensures valid ARO secret for Azure mirror with
// openshift images
// It also signals presense of Red Hat image registry keys in a
// cluster.status.RedHatKeysPresent field.

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

const (
	ControllerName = "PullSecret"

	controllerEnabled = "aro.pullsecret.enabled"
	controllerManaged = "aro.pullsecret.managed"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}
var rhKeys = []string{"registry.redhat.io", "cloud.openshift.com", "registry.connect.redhat.com"}

// Reconciler reconciles a Cluster object
type Reconciler struct {
	base.AROController

	secretsClient corev1client.SecretInterface
}

func NewReconciler(log *logrus.Entry, client client.Client, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},

		secretsClient: kubernetescli.CoreV1().Secrets(pullSecretName.Namespace),
	}
}

// Reconcile will make sure that the ACR part of the pull secret is correct. The
// conditions under which Reconcile is called are slightly unusual and are as
// follows:
//   - If the Cluster object changes, we'll see the *Cluster* object requested.
//   - If a Secret object owned by the Cluster object changes (e.g., but not
//     limited to, the configuration Secret, we'll see the *Cluster* object
//     requested).
//   - If the pull Secret object (which is not owned by the Cluster object)
//     changes, we'll see the pull Secret object requested.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.AROController.Client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.AROController.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.AROController.Log.Debug("running")
	userSecret, err := r.secretsClient.Get(ctx, pullSecretName.Name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	// reconcile global pull secret
	// detects if the global pull secret is broken and fixes it by using backup managed by ARO operator
	if instance.Spec.OperatorFlags.GetSimpleBoolean(controllerManaged) {
		operatorSecret := &corev1.Secret{}
		err = r.AROController.Client.Get(ctx, types.NamespacedName{Namespace: operator.Namespace, Name: operator.SecretName}, operatorSecret)
		if err != nil {
			return reconcile.Result{}, err
		}

		// fix pull secret if its broken to have at least the ARO pull secret
		userSecret, err = r.ensureGlobalPullSecret(ctx, operatorSecret, userSecret)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// reconcile cluster status
	// update the following information:
	// - list of Red Hat pull-secret keys in status.
	instance.Status.RedHatKeysPresent, err = r.parseRedHatKeys(userSecret)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.AROController.Client.Update(ctx, instance)
	return reconcile.Result{}, err
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	pullSecretPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return (o.GetName() == pullSecretName.Name && o.GetNamespace() == pullSecretName.Namespace) ||
			(o.GetName() == operator.SecretName && o.GetNamespace() == operator.Namespace)
	})

	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
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
		Named(ControllerName).
		Complete(r)
}

// ensureGlobalPullSecret checks the state of the pull secrets, in case of missing or broken ARO pull secret
// it replaces it with working one from controller Secret
// it takes care only for ARO pull secret, it does not touch the customer keys
func (r *Reconciler) ensureGlobalPullSecret(ctx context.Context, operatorSecret, userSecret *corev1.Secret) (secret *corev1.Secret, err error) {
	if operatorSecret == nil {
		return nil, errors.New("nil operator secret, cannot verify userData integrity")
	}

	// recreate := false
	secretData := make(map[string][]byte)

	if userSecret != nil {
		// when userSecret has broken type, recreates it with proper type
		// unfortunately the type field is immutable, therefore the whole secret have to be deleted and create once more
		if userSecret.Type != corev1.SecretTypeDockerConfigJson {
			err := r.AROController.Client.Delete(ctx, userSecret)
			if err != nil && !kerrors.IsNotFound(err) {
				return nil, err
			}
		}

		if json.Valid(userSecret.Data[corev1.DockerConfigJsonKey]) {
			secretData = userSecret.Data
		}
	}

	fixedData, update, err := pullsecret.Merge(string(secretData[corev1.DockerConfigJsonKey]), string(operatorSecret.Data[corev1.DockerConfigJsonKey]))
	if err != nil {
		return nil, err
	}

	if !update {
		return userSecret, nil
	}

	secretData[corev1.DockerConfigJsonKey] = []byte(fixedData)

	secretApplyConfig := applyv1.Secret(pullSecretName.Name, pullSecretName.Namespace).
		WithType(corev1.SecretTypeDockerConfigJson).
		WithData(secretData)
	applyOptions := metav1.ApplyOptions{FieldManager: ControllerName, Force: true}
	return r.secretsClient.Apply(ctx, secretApplyConfig, applyOptions)
}

// parseRedHatKeys unmarshal and extract following RH keys from pull-secret:
//   - redhat.registry.io
//   - cloud.openshift.com
//   - registry.connect.redhat.com
//
// if present, return error when the parsing fail, which means broken secret
func (r *Reconciler) parseRedHatKeys(secret *corev1.Secret) (foundKeys []string, err error) {
	// parse keys and validate JSON
	parsedKeys, err := pullsecret.UnmarshalSecretData(secret)
	if err != nil {
		r.AROController.Log.Info("pull secret is not valid json - recreating")
		return foundKeys, err
	}

	if parsedKeys != nil {
		for _, rhKey := range rhKeys {
			if v := parsedKeys[rhKey]; len(v) > 0 {
				foundKeys = append(foundKeys, rhKey)
			}
		}
	}

	return foundKeys, nil
}
