package openshiftcluster

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	machine "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset/typed/machine/v1beta1"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/util/restconfig"
)

type Manager struct {
	log        *logrus.Entry
	db         database.OpenShiftClusters
	authorizer autorest.Authorizer

	recordsets dns.RecordSetsClient
	groups     resources.GroupsClient

	machinesets machine.MachineSetInterface

	oc     *api.OpenShiftCluster
	domain string
}

func NewManager(log *logrus.Entry, db database.OpenShiftClusters, authorizer autorest.Authorizer, oc *api.OpenShiftCluster, domain string) (*Manager, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		log:        log,
		db:         db,
		authorizer: authorizer,

		oc:     oc,
		domain: domain,
	}

	m.recordsets = dns.NewRecordSetsClient(r.SubscriptionID)
	m.recordsets.Authorizer = authorizer

	m.groups = resources.NewGroupsClient(r.SubscriptionID)
	m.groups.Authorizer = authorizer
	m.groups.Client.PollingDuration = time.Hour

	restConfig, err := restconfig.RestConfig(oc.Properties.AdminKubeconfig)
	if err != nil {
		return nil, err
	}

	cli, err := machine.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	m.machinesets = cli.MachineSets("openshift-machine-api")

	return m, nil
}
