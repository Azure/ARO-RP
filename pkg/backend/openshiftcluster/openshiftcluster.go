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
	"github.com/jim-minter/rp/pkg/util/subnet"
)

type Manager struct {
	log          *logrus.Entry
	db           database.OpenShiftClusters
	fpAuthorizer autorest.Authorizer
	spAuthorizer autorest.Authorizer

	recordsets dns.RecordSetsClient
	groups     resources.GroupsClient

	subnets subnet.Manager

	doc    *api.OpenShiftClusterDocument
	domain string
}

func NewManager(log *logrus.Entry, db database.OpenShiftClusters, fpAuthorizer, spAuthorizer autorest.Authorizer, doc *api.OpenShiftClusterDocument, domain string) (*Manager, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		log:          log,
		db:           db,
		fpAuthorizer: fpAuthorizer,
		spAuthorizer: spAuthorizer,

		subnets: subnet.NewManager(r.SubscriptionID, spAuthorizer),

		doc:    doc,
		domain: domain,
	}

	m.recordsets = dns.NewRecordSetsClient(r.SubscriptionID)
	m.recordsets.Authorizer = fpAuthorizer

	m.groups = resources.NewGroupsClient(r.SubscriptionID)
	m.groups.Authorizer = fpAuthorizer
	m.groups.Client.PollingDuration = time.Hour

	return m, nil
}
