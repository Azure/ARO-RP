package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (f *frontend) getAdminOpenShiftClusterSerialConsole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._getAdminOpenShiftClusterSerialConsole(ctx, w, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _getAdminOpenShiftClusterSerialConsole(ctx context.Context, w http.ResponseWriter, r *http.Request, log *logrus.Entry) error {
	vars := mux.Vars(r)

	vmName := r.URL.Query().Get("vmName")
	err := validateAdminVMName(vmName)
	if err != nil {
		return err
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return err
	}

	resource, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return err
	}

	fpAuthorizer, err := f.env.FPAuthorizer(
		doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	s, err := newSerialTool(log, f.computeClientFactory(resource.SubscriptionID, fpAuthorizer),
		storage.NewAccountsClient(resource.SubscriptionID, fpAuthorizer), doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	return s.dump(ctx, vmName, w)
}

type serialTool struct {
	log *logrus.Entry
	oc  *api.OpenShiftCluster

	accounts        storage.AccountsClient
	virtualMachines compute.VirtualMachinesClient

	clusterResourceGroup string
}

func newSerialTool(log *logrus.Entry, virtualMachines compute.VirtualMachinesClient,
	accounts storage.AccountsClient, oc *api.OpenShiftCluster) (*serialTool, error) {

	return &serialTool{
		log: log,
		oc:  oc,

		accounts:        accounts,
		virtualMachines: virtualMachines,

		clusterResourceGroup: stringutils.LastTokenByte(oc.Properties.ClusterProfile.ResourceGroupID, '/'),
	}, nil
}

// dump writes to w the output of the serial console for vmname in raw bytes
func (s *serialTool) dump(ctx context.Context, vmname string, w http.ResponseWriter) error {
	vm, err := s.virtualMachines.Get(ctx, s.clusterResourceGroup, vmname, mgmtcompute.InstanceView)
	if err != nil {
		return err
	}

	u, err := url.Parse(*vm.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI)
	if err != nil {
		return err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 3 {
		return fmt.Errorf("serialConsoleLogBlobURI has %d parts, expected 3", len(parts))
	}

	t := time.Now().UTC().Truncate(time.Second)
	res, err := s.accounts.ListAccountSAS(ctx, s.clusterResourceGroup, "cluster"+s.oc.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
		Services:               mgmtstorage.B,
		ResourceTypes:          mgmtstorage.SignedResourceTypesO,
		Permissions:            mgmtstorage.R,
		Protocols:              mgmtstorage.HTTPS,
		SharedAccessStartTime:  &date.Time{Time: t},
		SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
	})
	if err != nil {
		return err
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return err
	}

	blobService := azstorage.NewAccountSASClient("cluster"+s.oc.Properties.StorageSuffix, v, azure.PublicCloud).GetBlobService()

	c := blobService.GetContainerReference(parts[1])

	b := c.GetBlobReference(parts[2])

	rc, err := b.Get(nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	w.Header().Add("Content-Type", "application/octet-stream")

	_, err = io.Copy(w, rc)
	return err
}
