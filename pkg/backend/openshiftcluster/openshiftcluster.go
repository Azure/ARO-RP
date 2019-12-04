package openshiftcluster

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
	"github.com/jim-minter/rp/pkg/util/azureclient/resources"
	"github.com/jim-minter/rp/pkg/util/subnet"
)

type Manager struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	fpAuthorizer autorest.Authorizer

	groups resources.GroupsClient

	subnets subnet.Manager

	doc *api.OpenShiftClusterDocument
}

func NewManager(log *logrus.Entry, env env.Interface, db database.OpenShiftClusters, fpAuthorizer autorest.Authorizer, doc *api.OpenShiftClusterDocument) (*Manager, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		log:          log,
		env:          env,
		db:           db,
		fpAuthorizer: fpAuthorizer,

		subnets: subnet.NewManager(r.SubscriptionID, fpAuthorizer),
		groups:  resources.NewGroupsClient(r.SubscriptionID, fpAuthorizer),

		doc: doc,
	}

	return m, nil
}
