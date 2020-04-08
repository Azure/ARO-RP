package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

type refreshableAuthorizer struct {
	autorest.Authorizer
	sp *adal.ServicePrincipalToken
}

func (ra *refreshableAuthorizer) Refresh() error {
	return ra.sp.Refresh()
}

type Dev interface {
	CreateARMResourceGroupRoleAssignment(context.Context, autorest.Authorizer, string) error
}

type dev struct {
	*prod

	log *logrus.Entry

	permissions     authorization.PermissionsClient
	roleassignments authorization.RoleAssignmentsClient
	applications    graphrbac.ApplicationsClient
	deployments     features.DeploymentsClient

	proxyPool       *x509.CertPool
	proxyClientCert []byte
	proxyClientKey  *rsa.PrivateKey
}

func newDev(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata) (*dev, error) {
	for _, key := range []string{
		"AZURE_ARM_CLIENT_ID",
		"AZURE_ARM_CLIENT_SECRET",
		"AZURE_FP_CLIENT_ID",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"DATABASE_NAME",
		"LOCATION",
		"PROXY_HOSTNAME",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	armAuthorizer, err := auth.NewClientCredentialsConfig(os.Getenv("AZURE_ARM_CLIENT_ID"), os.Getenv("AZURE_ARM_CLIENT_SECRET"), instancemetadata.TenantID()).Authorizer()
	if err != nil {
		return nil, err
	}

	d := &dev{
		log:             log,
		roleassignments: authorization.NewRoleAssignmentsClient(instancemetadata.SubscriptionID(), armAuthorizer),
	}

	d.prod, err = newProd(ctx, log, instancemetadata)
	if err != nil {
		return nil, err
	}
	d.prod.clustersGenevaLoggingEnvironment = "Test"
	d.prod.clustersGenevaLoggingConfigVersion = "2.3"

	fpGraphAuthorizer, err := d.FPAuthorizer(instancemetadata.TenantID(), azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	d.applications = graphrbac.NewApplicationsClient(instancemetadata.TenantID(), fpGraphAuthorizer)

	fpAuthorizer, err := d.FPAuthorizer(instancemetadata.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	d.permissions = authorization.NewPermissionsClient(instancemetadata.SubscriptionID(), fpAuthorizer)

	d.deployments = features.NewDeploymentsClient(instancemetadata.TenantID(), fpAuthorizer)

	b, err := ioutil.ReadFile("secrets/proxy.crt")
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, err
	}

	d.proxyPool = x509.NewCertPool()
	d.proxyPool.AddCert(cert)

	d.proxyClientCert, err = ioutil.ReadFile("secrets/proxy-client.crt")
	if err != nil {
		return nil, err
	}

	b, err = ioutil.ReadFile("secrets/proxy-client.key")
	if err != nil {
		return nil, err
	}

	d.proxyClientKey, err = x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *dev) InitializeAuthorizers() error {
	d.armClientAuthorizer = clientauthorizer.NewAll()
	d.adminClientAuthorizer = clientauthorizer.NewAll()
	return nil
}

func (d *dev) ACRName() string {
	return "arosvc"
}

func (d *dev) DatabaseName() string {
	return os.Getenv("DATABASE_NAME")
}

func (d *dev) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unimplemented network %q", network)
	}

	c, err := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext(ctx, network, os.Getenv("PROXY_HOSTNAME")+":443")
	if err != nil {
		return nil, err
	}

	c = tls.Client(c, &tls.Config{
		RootCAs: d.proxyPool,
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					d.proxyClientCert,
				},
				PrivateKey: d.proxyClientKey,
			},
		},
		ServerName: "proxy",
	})

	err = c.(*tls.Conn).Handshake()
	if err != nil {
		c.Close()
		return nil, err
	}

	r := bufio.NewReader(c)

	req, err := http.NewRequest(http.MethodConnect, "", nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	req.Host = address

	err = req.Write(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(r, req)
	if err != nil {
		c.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		c.Close()
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return &conn{Conn: c, r: r}, nil
}

func (d *dev) Listen() (net.Listener, error) {
	// in dev mode there is no authentication, so for safety we only listen on
	// localhost
	return net.Listen("tcp", "localhost:8443")
}

func (d *dev) FPAuthorizer(tenantID, resource string) (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, os.Getenv("AZURE_FP_CLIENT_ID"), d.fpCertificate, d.fpPrivateKey, resource)
	if err != nil {
		return nil, err
	}

	return &refreshableAuthorizer{autorest.NewBearerAuthorizer(sp), sp}, nil
}

func (d *dev) MetricsSocketPath() string {
	return "mdm_statsd.socket"
}

func (d *dev) CreateARMResourceGroupRoleAssignment(ctx context.Context, fpAuthorizer autorest.Authorizer, resourceGroup string) error {
	d.log.Print("development mode: applying resource group role assignment")

	res, err := d.applications.GetServicePrincipalsIDByAppID(ctx, os.Getenv("AZURE_FP_CLIENT_ID"))
	if err != nil {
		return err
	}

	_, err = d.roleassignments.Create(ctx, "/subscriptions/"+d.SubscriptionID()+"/resourceGroups/"+resourceGroup, uuid.NewV4().String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + d.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"),
			PrincipalID:      res.Value,
		},
	})
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "RoleAssignmentExists" {
			err = nil
		}
	}
	if err != nil {
		return err
	}

	// Issue: https://github.com/Azure/ARO-RP/issues/31
	// rbac client returns right permissions, but access is not yet propagated
	// in the azure backends. We test by trying to call API directly and check if
	// role was applied.
	d.log.Print("development mode: refreshing authorizer")
	err = fpAuthorizer.(*refreshableAuthorizer).Refresh()
	if err != nil {
		return err
	}

	return wait.Poll(time.Second, time.Minute, func() (bool, error) {
		// this should always error. Either 403 or 404
		_, err := d.deployments.Get(ctx, resourceGroup, "dummy")
		if detailedErr, ok := err.(autorest.DetailedError); ok {
			if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
				requestErr.ServiceError != nil &&
				requestErr.ServiceError.Code == "AuthorizationFailed" {
				return false, nil
			}
		}
		return true, nil
	})
}
