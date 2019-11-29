package openshiftcluster

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
)

type Manager struct {
	log        *logrus.Entry
	db         database.OpenShiftClusters
	authorizer autorest.Authorizer

	recordsets dns.RecordSetsClient
	groups     resources.GroupsClient

	doc    *api.OpenShiftClusterDocument
	domain string
}

func NewManager(log *logrus.Entry, db database.OpenShiftClusters, authorizer autorest.Authorizer, doc *api.OpenShiftClusterDocument, domain string) (*Manager, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		log:        log,
		db:         db,
		authorizer: authorizer,

		doc:    doc,
		domain: domain,
	}

	m.recordsets = dns.NewRecordSetsClient(r.SubscriptionID)
	m.recordsets.Authorizer = authorizer

	m.groups = resources.NewGroupsClient(r.SubscriptionID)
	m.groups.Authorizer = authorizer
	m.groups.Client.PollingDuration = time.Hour

	return m, nil
}
