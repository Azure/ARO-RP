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
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/operator/api/v1alpha1"
	"github.com/Azure/ARO-RP/operator/controllers/consts"
	"github.com/Azure/ARO-RP/operator/controllers/deploy"
	"github.com/Azure/ARO-RP/operator/controllers/statusreporter"
)

// InternetChecker reconciles a Cluster object
type InternetChecker struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

func (r *InternetChecker) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	operatorNs, err := deploy.OperatorNamespace()
	if err != nil {
		r.Log.Error(err, "deploy.OperatorNamespace")
		return consts.ReconcileResultError, err
	}

	if request.Name != arov1alpha1.SingletonClusterName || request.Namespace != operatorNs {
		return consts.ReconcileResultIgnore, nil
	}
	r.Log.Info("Polling outgoing internet connection")

	// TODO https://github.com/Azure/OpenShift/issues/185

	req, err := http.NewRequest("GET", "https://management.azure.com", nil)
	if err != nil {
		r.Log.Error(err, "failed building request")
		return consts.ReconcileResultError, err
	}
	req.Header.Set("Content-Type", "application/json")

	ctx := context.TODO()
	sr := statusreporter.NewStatusReporter(r.Client, request.Namespace, request.Name)
	client := &http.Client{}
	resp, err := client.Do(req)
	r.Log.Info("response", "code", resp.Status, "err", err)
	if err != nil || resp.StatusCode != http.StatusOK {
		err = sr.SetNoInternetConnection(ctx, err)
	} else {
		err = sr.SetInternetConnected(ctx)
	}
	if err != nil {
		r.Log.Error(err, "StatusReporter")
		return consts.ReconcileResultError, err
	}

	r.Log.Info("done, requeueing")
	return consts.ReconcileResultRequeue, nil
}

func (r *InternetChecker) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Complete(r)
}
