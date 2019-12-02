package frontend

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

type noContent struct{}

func (noContent) Error() string { return "" }

func (f *frontend) deleteOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)

	_, err := f.db.OpenShiftClusters.Patch(api.Key(r.URL.Path), func(doc *api.OpenShiftClusterDocument) error {
		return f._deleteOpenShiftCluster(doc)
	})

	reply(log, w, nil, err)
}

func (f *frontend) _deleteOpenShiftCluster(doc *api.OpenShiftClusterDocument) error {
	if doc == nil {
		return &noContent{}
	}

	_, err := f.validateSubscriptionState(doc.Key, api.SubscriptionStateRegistered, api.SubscriptionStateWarned, api.SubscriptionStateSuspended)
	if err != nil {
		return err
	}

	err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
	if err != nil {
		return err
	}

	doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateDeleting
	doc.Dequeues = 0

	return nil
}
