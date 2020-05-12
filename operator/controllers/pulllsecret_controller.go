/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/operator/api/v1alpha1"
	"github.com/Azure/ARO-RP/operator/controllers/consts"
	"github.com/Azure/ARO-RP/operator/controllers/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullsecretReconciler reconciles a Cluster object
type PullsecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

func (r *PullsecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.NamespacedName != pullSecretName {
		// filter out other secrets.
		return consts.ReconcileResultIgnore, nil
	}

	r.Log.Info("Reconciling pull-secret")

	ctx := context.TODO()
	isCreate := false
	ps := &corev1.Secret{}
	err := r.Client.Get(ctx, request.NamespacedName, ps)
	if err != nil && errors.IsNotFound(err) {
		isCreate = true
		ps = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
	} else if err != nil {
		r.Log.Error(err, "failed to Get pull secret")
		return consts.ReconcileResultError, err
	}

	changed, err := r.pullSecretRepair(ps)
	if err != nil {
		return consts.ReconcileResultError, err
	}
	if !isCreate && !changed {
		r.Log.Info("Skip reconcile: Pull Secret repair not required")
		return consts.ReconcileResultDone, nil
	}
	if isCreate {
		r.Log.Info("Re-creating the Pull Secret")
		err = r.Client.Create(ctx, ps)
	} else if changed {
		r.Log.Info("Updating the Pull Secret")
		err = r.Client.Update(ctx, ps)
	}
	if err != nil {
		r.Log.Error(err, "Failed to repair the Pull Secret")
		return consts.ReconcileResultError, err
	}
	r.Log.Info("done, requeueing")
	return consts.ReconcileResultDone, nil
}

func (r *PullsecretReconciler) pullSecretRepair(cr *corev1.Secret) (bool, error) {
	if cr.Data == nil {
		cr.Data = map[string][]byte{}
	}

	// The idea here is you mount a secret as a file under /pull-secrets with
	// the same name as the registry in the pull secret.
	psPath := "/pull-secrets"
	pathOverride := os.Getenv("PULL_SECRET_PATH") // for development
	if pathOverride != "" {
		psPath = pathOverride
	}

	newPS, changed, err := pullsecret.Repair(cr.Data[corev1.DockerConfigJsonKey], psPath)
	if err != nil {
		return false, err
	}
	if changed {
		cr.Data[corev1.DockerConfigJsonKey] = newPS
	}
	return changed, nil
}

func (r *PullsecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Complete(r)
}
