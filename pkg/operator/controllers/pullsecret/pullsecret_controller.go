package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	aro "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullSecretReconciler reconciles a Cluster object
type PullSecretReconciler struct {
	kubernetescli kubernetes.Interface
	arocli        aroclient.Interface
	samplescli    samplesclient.Interface
	log           *logrus.Entry
}

type PullSecretAction int

const (
	NoAction           PullSecretAction = iota
	CreatePullSecret   PullSecretAction = iota
	RecreatePullSecret PullSecretAction = iota
	UpdatePullSecret   PullSecretAction = iota
)

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.Interface, samplescli samplesclient.Interface) *PullSecretReconciler {
	return &PullSecretReconciler{
		log:           log,
		kubernetescli: kubernetescli,
		arocli:        arocli,
		samplescli:    samplescli,
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
	if request.NamespacedName != pullSecretName &&
		request.Name != arov1alpha1.SingletonClusterName {
		return reconcile.Result{}, nil
	}

	mysec, err := r.kubernetescli.CoreV1().Secrets(operator.Namespace).Get(ctx, operator.SecretName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// cluster object is needed, controller changes its state based on RH pull secret presence
	cluster, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ps, err := r.pullsecret(ctx)
		if err != nil {
			return err
		}

		// fix pull secret if its broken to have at least the ARO pull secret
		secret, action, err := r.updateGlobalPullSecret(mysec, ps)
		if err != nil {
			return err
		}
		err = r.emitGlobalPullSecretChange(ctx, secret, action)
		if err != nil {
			return err
		}

		// change the condition of the operator based on the Red Hat key presence
		redHatKeyCondition, err := r.updateRedHatKeyCondition(secret)
		if err != nil {
			return err
		}
		err = controllers.SetCondition(ctx, r.arocli, redHatKeyCondition, operator.RoleMaster)
		if err != nil {
			return err
		}

		if cluster.Spec.Features.ManageSamplesOperator {

			updatedSamplesOperator, err := r.emitSamplesControllerChange(ctx, redHatKeyCondition)
			if err != nil {
				return nil
			}
			err = controllers.SetCondition(ctx, r.arocli, updatedSamplesOperator, operator.RoleMaster)
			if err != nil {
				return err
			}

		}
		return nil
	})
}

func (r *PullSecretReconciler) pullsecret(ctx context.Context) (*v1.Secret, error) {
	ps, err := r.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Get(ctx, pullSecretName.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ps, nil
}

func triggerReconcile(secret *v1.Secret) bool {
	return (secret.Name == pullSecretName.Name && secret.Namespace == pullSecretName.Namespace) ||
		(secret.Name == operator.SecretName && secret.Namespace == operator.Namespace)
}

// SetupWithManager setup our manager
func (r *PullSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	isPullSecret := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, ok := e.ObjectOld.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			secret, ok := e.ObjectOld.(*v1.Secret)
			return ok && triggerReconcile(secret)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			secret, ok := e.Object.(*v1.Secret)
			return ok && triggerReconcile(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, ok := e.Object.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			secret, ok := e.Object.(*v1.Secret)
			return ok && triggerReconcile(secret)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to For(&v1.Secret{})., but can't call For multiple times on one builder
		Watches(&source.Kind{Type: &v1.Secret{}}, &handler.EnqueueRequestForObject{}).
		Owns(&v1.Secret{}).
		WithEventFilter(isPullSecret).
		Named(controllers.PullSecretControllerName).
		Complete(r)
}

// updateGlobalPullSecret checks the state of the pull secrets, in case of missing or broken ARO pull secret
// it replaces it with working one from controller Secret
func (r *PullSecretReconciler) updateGlobalPullSecret(operatorSecret, userSecret *v1.Secret) (updatedSecret *v1.Secret, action PullSecretAction, err error) {
	if operatorSecret == nil {
		return nil, NoAction, errors.New("Nil operator secret, cannot verify userData integrity")
	}

	operatorData, err := pullsecret.UnmarshalSecretData(operatorSecret)
	if err != nil {
		// no reference data cannot verify integrity of the data
		return nil, NoAction, errors.New("Cannot parse operatorSecret, cannot verify userSecret integrity")
	}

	var secret *v1.Secret
	create := false

	if userSecret == nil {
		action = CreatePullSecret
		create = true
	}

	// userSecret can happen to have broken type, which means it have to be recreated
	// with proper type
	if userSecret != nil && userSecret.Type != v1.SecretTypeDockerConfigJson {
		action = RecreatePullSecret
		create = true
	}

	if create {
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: v1.SecretTypeDockerConfigJson,
		}
	} else {
		secret = userSecret.DeepCopy()
	}

	userData, err := pullsecret.UnmarshalSecretData(secret)
	if err != nil || userData == nil {
		// cannot parse data it is broken, recreate the ARO content
		userData = &pullsecret.SerializedAuthMap{}
		secret.Data = make(map[string][]byte)
	}

	fixedData, update := pullsecret.FixPullSecretData(operatorData, userData)

	if !create && !update {
		return secret, NoAction, nil
	}

	if !create && update {
		action = UpdatePullSecret
	}

	rawFixedData, err := json.Marshal(fixedData)
	if err != nil {
		return nil, NoAction, err
	}

	secret.Data[v1.DockerConfigJsonKey] = rawFixedData
	return secret, action, nil
}

// emitGlobalPullSecretChange performs update of the secret when the user pull secret is broken or require recreation
func (r *PullSecretReconciler) emitGlobalPullSecretChange(ctx context.Context, secret *v1.Secret, action PullSecretAction) error {
	if action == NoAction {
		// no action required
		return nil
	}

	if action == RecreatePullSecret {
		// unfortunately the type field is immutable.
		err := r.kubernetescli.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	if action == CreatePullSecret || action == RecreatePullSecret {
		r.log.Info("re-creating pull secret")
		_, err := r.kubernetescli.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})

		return err
	}

	r.log.Info("updating pull secret")
	_, err := r.kubernetescli.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func (r *PullSecretReconciler) updateRedHatKeyCondition(secret *v1.Secret) (*status.Condition, error) {
	// parse keys and validate JSON
	parsedKeys, err := pullsecret.UnmarshalSecretData(secret)
	foundKey := false
	failed := false
	if err != nil {
		r.log.Info("pull secret is not valid json - recreating")
		delete(secret.Data, v1.DockerConfigJsonKey)
		failed = true
	} else {
		foundKey = r.checkRHRegistryKeys(parsedKeys)
	}

	keyCondition := r.keyCondition(failed, foundKey)
	return keyCondition, nil
}

func (r *PullSecretReconciler) emitSamplesControllerChange(ctx context.Context, state *status.Condition) (*status.Condition, error) {
	updated := false
	enable := false
	if state.Type == aro.RedHatKeyPresent && state.IsTrue() {
		enable = true
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c, err := r.samplescli.SamplesV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if c.Spec.SamplesRegistry != "" {
			// if the samples registry point elsewhere no action
			return nil
		}

		oldManagementState := c.Spec.ManagementState

		if enable {
			c.Spec.ManagementState = operatorv1.Managed
		} else {
			c.Spec.ManagementState = operatorv1.Removed
		}

		if oldManagementState == c.Spec.ManagementState {
			return nil
		}

		_, err = r.samplescli.SamplesV1().Configs().Update(ctx, c, metav1.UpdateOptions{})
		if err == nil {
			updated = true
		}
		return err
	})

	samplesCondition := r.samplesCondition(updated, enable)
	return samplesCondition, err
}

// checkRHRegistryKeys checks whether the rhRegistry keys:
//   - redhat.registry.io
// are present in the pullSecret
func (r *PullSecretReconciler) checkRHRegistryKeys(psData *pullsecret.SerializedAuthMap) (foundKey bool) {
	rhKeys := []string{
		"registry.redhat.io",
	}

	if psData == nil {
		return foundKey
	}

	for _, key := range rhKeys {
		if auth, ok := psData.Auths[key]; ok && len(auth.Auth) > 0 {
			foundKey = true
		}
	}

	return foundKey
}

func (r *PullSecretReconciler) keyCondition(failed bool, foundKey bool) *status.Condition {
	keyCondition := &status.Condition{
		Type:   aro.RedHatKeyPresent,
		Status: v1.ConditionFalse,
		Reason: "CheckFailed",
	}

	if failed {
		keyCondition.Message = "Cannot parse pull-secret"
		return keyCondition
	}

	keyCondition.Reason = "CheckDone"

	if !foundKey {
		keyCondition.Message = "No Red Hat key found in pull-secret"
		return keyCondition
	}

	keyCondition.Status = v1.ConditionTrue
	keyCondition.Message = "Red Hat registry key present in pull-secret"

	return keyCondition
}

// samplesCondition indicates whether aroOperator modified state of the cluster-samples-operator
func (r *PullSecretReconciler) samplesCondition(updated bool, foundKey bool) *status.Condition {
	samplesCondition := &status.Condition{
		Type:   aro.SamplesOperatorEnabled,
		Status: v1.ConditionFalse,
		Reason: "RedHatKey",
	}
	sb := strings.Builder{}
	sb.WriteString("cluster-samples-operator ")

	if updated {
		sb.WriteString("updated to ")
	} else {
		sb.WriteString("in ")
	}

	if foundKey {
		sb.WriteString("managed ")
		samplesCondition.Status = v1.ConditionTrue
	} else {
		sb.WriteString("removed ")
	}

	sb.WriteString("state")
	samplesCondition.Message = sb.String()

	return samplesCondition
}
