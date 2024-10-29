package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/go-multierror"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	utilkubernetes "github.com/Azure/ARO-RP/pkg/util/kubernetes"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

//go:embed staticresources
var embeddedFiles embed.FS

type Operator interface {
	Install(context.Context) error
	Update(context.Context) error
	CreateOrUpdateCredentialsRequest(context.Context) error
	IsReady(context.Context) (bool, error)
	Restart(context.Context, []string) error
	IsRunningDesiredVersion(context.Context) (bool, error)
	RenewMDSDCertificate(context.Context) error
	EnsureUpgradeAnnotation(context.Context) error
	SyncClusterObject(context.Context) error
	SetForceReconcile(context.Context, bool) error
}

type operator struct {
	log             *logrus.Entry
	env             env.Interface
	oc              *api.OpenShiftCluster
	subscriptiondoc *api.SubscriptionDocument

	arocli        aroclient.Interface
	client        clienthelper.Interface
	extensionscli extensionsclient.Interface
	kubernetescli kubernetes.Interface
	operatorcli   operatorclient.Interface
	dh            dynamichelper.Interface
}

func New(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, arocli aroclient.Interface, client clienthelper.Interface, extensionscli extensionsclient.Interface, kubernetescli kubernetes.Interface, operatorcli operatorclient.Interface) (Operator, error) {
	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}
	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return nil, err
	}

	return &operator{
		log: log,
		env: env,
		oc:  oc,

		arocli:          arocli,
		client:          client,
		extensionscli:   extensionscli,
		kubernetescli:   kubernetescli,
		operatorcli:     operatorcli,
		dh:              dh,
		subscriptiondoc: subscriptionDoc,
	}, nil
}

type deploymentData struct {
	Image                        string
	Version                      string
	IsLocalDevelopment           bool
	SupportsPodSecurityAdmission bool
	UsesWorkloadIdentity         bool
	TokenVolumeMountPath         string
	FederatedTokenFilePath       string
}

func (o *operator) SetForceReconcile(ctx context.Context, enable bool) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		c, err := o.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if enable {
			c.Spec.OperatorFlags[pkgoperator.ForceReconciliation] = "true"
		} else {
			c.Spec.OperatorFlags[pkgoperator.ForceReconciliation] = "false"
		}
		_, err = o.arocli.AroV1alpha1().Clusters().Update(ctx, c, metav1.UpdateOptions{})
		return err
	})
}

func templateManifests(data deploymentData) ([][]byte, error) {
	templatesRoot, err := template.ParseFS(embeddedFiles, "staticresources/*.yaml")
	if err != nil {
		return nil, err
	}
	templatesMaster, err := template.ParseFS(embeddedFiles, "staticresources/master/*")
	if err != nil {
		return nil, err
	}
	templatesWorker, err := template.ParseFS(embeddedFiles, "staticresources/worker/*")
	if err != nil {
		return nil, err
	}

	templatedFiles := make([][]byte, 0)
	templatesArray := []*template.Template{templatesRoot, templatesMaster, templatesWorker}

	for _, templates := range templatesArray {
		for _, templ := range templates.Templates() {
			buff := &bytes.Buffer{}
			if err := templ.Execute(buff, data); err != nil {
				return nil, err
			}
			templatedFiles = append(templatedFiles, buff.Bytes())
		}
	}
	return templatedFiles, nil
}

func (o *operator) createDeploymentData(ctx context.Context) (deploymentData, error) {
	image := o.env.AROOperatorImage()

	// HACK: Override for ARO_IMAGE env variable setup in local-dev mode
	version := "latest"
	if strings.Contains(image, ":") {
		str := strings.Split(image, ":")
		version = str[len(str)-1]
	}

	// Set version correctly if it's overridden
	if o.oc.Properties.OperatorVersion != "" {
		version = o.oc.Properties.OperatorVersion
		image = fmt.Sprintf("%s/aro:%s", o.env.ACRDomain(), version)
	}

	// Set Pod Security Admission parameters if > 4.10
	// this only gets set on PUCM (Everything or OperatorUpdate)
	usePodSecurityAdmission, err := pkgoperator.ShouldUsePodSecurityStandard(ctx, o.client)
	if err != nil {
		return deploymentData{}, err
	}

	data := deploymentData{
		IsLocalDevelopment:           o.env.IsLocalDevelopmentMode(),
		Image:                        image,
		SupportsPodSecurityAdmission: usePodSecurityAdmission,
		Version:                      version,
	}

	if o.oc.UsesWorkloadIdentity() {
		data.UsesWorkloadIdentity = o.oc.UsesWorkloadIdentity()
		data.TokenVolumeMountPath = filepath.Dir(pkgoperator.OperatorTokenFile)
		data.FederatedTokenFilePath = pkgoperator.OperatorTokenFile
	}

	return data, nil
}

func (o *operator) createObjects(ctx context.Context) ([]kruntime.Object, error) {
	deploymentData, err := o.createDeploymentData(ctx)
	if err != nil {
		return nil, err
	}

	templated, err := templateManifests(deploymentData)
	if err != nil {
		return nil, err
	}
	objects := make([]kruntime.Object, 0, len(templated))
	for _, v := range templated {
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(v, nil, nil)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

func (o *operator) resources(ctx context.Context) ([]kruntime.Object, error) {
	// first static resources from Assets
	results, err := o.createObjects(ctx)
	if err != nil {
		return nil, err
	}

	// then dynamic resources
	if o.oc.UsesWorkloadIdentity() {
		operatorIdentitySecret, err := o.generateOperatorIdentitySecret()
		if err != nil {
			return nil, err
		}

		results = append(results, operatorIdentitySecret)
	}

	key, cert := o.env.ClusterGenevaLoggingSecret()
	gcsKeyBytes, err := utilpem.Encode(key)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := utilpem.Encode(cert)
	if err != nil {
		return nil, err
	}

	ps, err := pullsecret.Build(o.oc, "")
	if err != nil {
		return nil, err
	}

	// create a secret here for genevalogging, later we will copy it to
	// the genevalogging namespace.
	return append(results,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgoperator.SecretName,
				Namespace: pkgoperator.Namespace,
			},
			Data: map[string][]byte{
				genevalogging.GenevaCertName: gcsCertBytes,
				genevalogging.GenevaKeyName:  gcsKeyBytes,
				corev1.DockerConfigJsonKey:   []byte(ps),
			},
		},
	), nil
}

func (o *operator) generateOperatorIdentitySecret() (*corev1.Secret, error) {
	var operatorIdentity *api.PlatformWorkloadIdentity // use a pointer to make it easy to check if we found an identity below
	for k, i := range o.oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		if k == pkgoperator.OperatorIdentityName {
			operatorIdentity = &i
			break
		}
	}

	if operatorIdentity == nil {
		return nil, fmt.Errorf("operator identity %s not found", pkgoperator.OperatorIdentityName)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pkgoperator.Namespace,
			Name:      pkgoperator.OperatorIdentitySecretName,
		},
		StringData: map[string]string{
			"azure_client_id":            operatorIdentity.ClientID,
			"azure_tenant_id":            o.subscriptiondoc.Subscription.Properties.TenantID,
			"azure_region":               o.oc.Location,
			"azure_subscription_id":      o.subscriptiondoc.ID,
			"azure_federated_token_file": pkgoperator.OperatorTokenFile,
		},
	}, nil
}

func (o *operator) clusterObject() (*arov1alpha1.Cluster, error) {
	vnetID, _, err := apisubnet.Split(o.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}

	domain := o.oc.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(domain, '.') {
		domain += "." + o.env.Domain()
	}

	ingressIP, err := checkIngressIP(o.oc.Properties.IngressProfiles)
	if err != nil {
		return nil, err
	}

	serviceSubnets := []string{
		"/subscriptions/" + o.env.SubscriptionID() + "/resourceGroups/" + o.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet",
		"/subscriptions/" + o.env.SubscriptionID() + "/resourceGroups/" + o.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-vnet/subnets/rp-subnet",
	}

	// Avoiding issues with dev environment when gateway is not present
	if o.oc.Properties.FeatureProfile.GatewayEnabled {
		serviceSubnets = append(serviceSubnets, "/subscriptions/"+o.env.SubscriptionID()+"/resourceGroups/"+o.env.GatewayResourceGroup()+"/providers/Microsoft.Network/virtualNetworks/gateway-vnet/subnets/gateway-subnet")
	}

	cluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			ResourceID:             o.oc.ID,
			ClusterResourceGroupID: o.oc.Properties.ClusterProfile.ResourceGroupID,
			Domain:                 domain,
			ACRDomain:              o.env.ACRDomain(),
			AZEnvironment:          o.env.Environment().Name,
			Location:               o.env.Location(),
			InfraID:                o.oc.Properties.InfraID,
			ArchitectureVersion:    int(o.oc.Properties.ArchitectureVersion),
			VnetID:                 vnetID,
			StorageSuffix:          o.oc.Properties.StorageSuffix,
			GenevaLogging: arov1alpha1.GenevaLoggingSpec{
				ConfigVersion:            o.env.ClusterGenevaLoggingConfigVersion(),
				MonitoringGCSAccount:     o.env.ClusterGenevaLoggingAccount(),
				MonitoringGCSEnvironment: o.env.ClusterGenevaLoggingEnvironment(),
				MonitoringGCSNamespace:   o.env.ClusterGenevaLoggingNamespace(),
			},
			ServiceSubnets: serviceSubnets,
			InternetChecker: arov1alpha1.InternetCheckerSpec{
				URLs: []string{
					fmt.Sprintf("https://%s/", o.env.ACRDomain()),
					o.env.Environment().ActiveDirectoryEndpoint,
					o.env.Environment().ResourceManagerEndpoint,
					o.env.Environment().GenevaMonitoringEndpoint,
				},
			},

			APIIntIP:                 o.oc.Properties.APIServerProfile.IntIP,
			IngressIP:                ingressIP,
			GatewayPrivateEndpointIP: o.oc.Properties.NetworkProfile.GatewayPrivateEndpointIP,
			// Update the OperatorFlags from the version in the RP
			OperatorFlags: arov1alpha1.OperatorFlags(o.oc.Properties.OperatorFlags),
		},
	}

	if o.oc.Properties.FeatureProfile.GatewayEnabled && o.oc.Properties.NetworkProfile.GatewayPrivateEndpointIP != "" {
		cluster.Spec.GatewayDomains = append(o.env.GatewayDomains(), o.oc.Properties.ImageRegistryStorageAccountName+".blob."+o.env.Environment().StorageEndpointSuffix)
	} else {
		// covers the case of an admin-disable, we need to update dnsmasq on each node
		cluster.Spec.GatewayDomains = make([]string, 0)
	}
	return cluster, nil
}

func (o *operator) SyncClusterObject(ctx context.Context) error {
	resource, err := o.clusterObject()
	if err != nil {
		return err
	}
	return o.dh.Ensure(ctx, resource)
}

func (o *operator) Install(ctx context.Context) error {
	resources, err := o.resources(ctx)
	if err != nil {
		return err
	}

	// If we're installing the Operator for the first time, include the Cluster
	// object, otherwise it is updated separately
	cluster, err := o.clusterObject()
	if err != nil {
		return err
	}
	resources = append(resources, cluster)

	return o.applyDeployment(ctx, resources)
}

func (o *operator) Update(ctx context.Context) error {
	resources, err := o.resources(ctx)
	if err != nil {
		return err
	}
	return o.applyDeployment(ctx, resources)
}

func (o *operator) applyDeployment(ctx context.Context, resources []kruntime.Object) error {
	err := dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		err = o.dh.Ensure(ctx, resource)
		if err != nil {
			return err
		}

		gvks, _, err := scheme.Scheme.ObjectKinds(resource)
		if err != nil {
			return err
		}

		switch gvks[0].GroupKind().String() {
		case "CustomResourceDefinition.apiextensions.k8s.io":
			acc, err := meta.Accessor(resource)
			if err != nil {
				return err
			}

			err = wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
				crd, err := o.extensionscli.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, acc.GetName(), metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				return isCRDEstablished(crd), nil
			})
			if err != nil {
				return err
			}

			err = o.dh.Refresh()
			if err != nil {
				return err
			}

		case "Cluster.aro.openshift.io":
			// add an owner reference onto our configuration secret.  This is
			// can only be done once we've got the cluster UID.  It is needed to
			// ensure that secret updates trigger updates of the appropriate
			// controllers
			err = retry.OnError(wait.Backoff{
				Steps:    60,
				Duration: time.Second,
			}, func(err error) bool {
				// IsForbidden here is intended to catch the following transient
				// error: secrets "cluster" is forbidden: cannot set
				// blockOwnerDeletion in this case because cannot find
				// RESTMapping for APIVersion aro.openshift.io/v1alpha1 Kind
				// Cluster: no matches for kind "Cluster" in version
				// "aro.openshift.io/v1alpha1"
				return kerrors.IsForbidden(err) || kerrors.IsConflict(err)
			}, func() error {
				cluster, err := o.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				s, err := o.kubernetescli.CoreV1().Secrets(pkgoperator.Namespace).Get(ctx, pkgoperator.SecretName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				err = controllerutil.SetControllerReference(cluster, s, scheme.Scheme)
				if err != nil {
					return err
				}

				_, err = o.kubernetescli.CoreV1().Secrets(pkgoperator.Namespace).Update(ctx, s, metav1.UpdateOptions{})
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateOrUpdateCredentialsRequest just creates/updates the ARO operator's CredentialsRequest
// rather than doing all of the operator's associated Kubernetes resources.
func (o *operator) CreateOrUpdateCredentialsRequest(ctx context.Context) error {
	templ, err := template.ParseFS(embeddedFiles, "staticresources/credentialsrequest.yaml")
	if err != nil {
		return err
	}

	buff := &bytes.Buffer{}
	err = templ.Execute(buff, nil)
	if err != nil {
		return err
	}

	crUnstructured, err := dynamichelper.DecodeUnstructured(buff.Bytes())
	if err != nil {
		return err
	}

	return o.dh.Ensure(ctx, crUnstructured)
}

func (o *operator) RenewMDSDCertificate(ctx context.Context) error {
	key, cert := o.env.ClusterGenevaLoggingSecret()
	gcsKeyBytes, err := utilpem.Encode(key)
	if err != nil {
		return err
	}
	gcsCertBytes, err := utilpem.Encode(cert)
	if err != nil {
		return err
	}

	s, err := o.kubernetescli.CoreV1().Secrets(pkgoperator.Namespace).Get(ctx, pkgoperator.SecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	s.Data["gcscert.pem"] = gcsCertBytes
	s.Data["gcskey.pem"] = gcsKeyBytes

	_, err = o.kubernetescli.CoreV1().Secrets(pkgoperator.Namespace).Update(ctx, s, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (o *operator) EnsureUpgradeAnnotation(ctx context.Context) error {
	if !o.oc.UsesWorkloadIdentity() {
		return nil
	}

	upgradeableTo := string(*o.oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo)
	upgradeableAnnotation := "cloudcredential.openshift.io/upgradeable-to"

	cloudcredentialobject, err := o.operatorcli.OperatorV1().CloudCredentials().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	if cloudcredentialobject.Annotations == nil {
		cloudcredentialobject.Annotations = map[string]string{}
	}

	cloudcredentialobject.Annotations[upgradeableAnnotation] = upgradeableTo

	_, err = o.operatorcli.OperatorV1().CloudCredentials().Update(ctx, cloudcredentialobject, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (o *operator) IsReady(ctx context.Context) (bool, error) {
	deploymentOk := true
	var deploymentErr error

	deployments := o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace)
	replicasets := o.kubernetescli.AppsV1().ReplicaSets(pkgoperator.Namespace)
	pods := o.kubernetescli.CoreV1().Pods(pkgoperator.Namespace)

	for _, deployment := range []string{"aro-operator-master", "aro-operator-worker"} {
		ok, err := ready.CheckDeploymentIsReady(ctx, deployments, deployment)()
		o.log.Infof("deployment %q ok status is: %v, err is: %v", deployment, ok, err)
		deploymentOk = deploymentOk && ok
		if deploymentErr == nil && err != nil {
			deploymentErr = err
		}
		if ok {
			continue
		}

		d, err := deployments.Get(ctx, deployment, metav1.GetOptions{})
		if err != nil {
			o.log.Errorf("failed to get deployment %q: %s", deployment, err)
			continue
		}
		j, err := json.Marshal(d.Status)
		if err != nil {
			o.log.Errorf("failed to serialize deployment %q: %s", deployment, err)
			continue
		}
		o.log.Infof("deployment %q status: %s", deployment, string(j))

		// Gather and print status of this deployment's replicasets
		rs, err := replicasets.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", deployment)})
		if err != nil {
			o.log.Errorf("failed to list replicasets: %s", err)
			continue
		}
		for _, replicaset := range rs.Items {
			r, err := replicasets.Get(ctx, replicaset.Name, metav1.GetOptions{})
			if err != nil {
				o.log.Errorf("failed to get replicaset %s: %s", replicaset.Name, err)
				continue
			}
			j, err := json.Marshal(r.Status)
			if err != nil {
				o.log.Errorf("failed to serialize replicaset status %q: %s", replicaset.Name, err)
				continue
			}
			o.log.Infof("replicaset %q status: %s", replicaset.Name, string(j))
		}

		// Gather and print status of this deployment's pods
		ps, err := pods.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", deployment)})
		if err != nil {
			o.log.Errorf("failed to list pods: %s", err)
			continue
		}
		for _, pod := range ps.Items {
			p, err := pods.Get(ctx, pod.Name, metav1.GetOptions{})
			if err != nil {
				o.log.Errorf("failed to get pod %s: %s", pod.Name, err)
				continue
			}
			j, err := json.Marshal(p.Status)
			if err != nil {
				o.log.Errorf("failed to serialize pod status %q: %s", pod.Name, err)
				continue
			}
			o.log.Infof("pod %q status: %s", pod.Name, string(j))
		}
	}
	return deploymentOk, deploymentErr
}

func (o *operator) Restart(ctx context.Context, deploymentNames []string) error {
	var result error
	for _, dn := range deploymentNames {
		err := utilkubernetes.Restart(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), pkgoperator.Namespace, dn)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

func checkOperatorDeploymentVersion(ctx context.Context, cli appsv1client.DeploymentInterface, name string, desiredVersion string) (bool, error) {
	d, err := cli.Get(ctx, name, metav1.GetOptions{})
	switch {
	case kerrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}
	if d.Labels["version"] != desiredVersion {
		return false, nil
	}
	return true, nil
}

func checkPodImageVersion(ctx context.Context, cli corev1client.PodInterface, role string, desiredVersion string) (bool, error) {
	podList, err := cli.List(ctx, metav1.ListOptions{LabelSelector: "app=" + role})
	switch {
	case kerrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}
	imageTag := "latest"
	for _, pod := range podList.Items {
		if strings.Contains(pod.Spec.Containers[0].Image, ":") {
			str := strings.Split(pod.Spec.Containers[0].Image, ":")
			imageTag = str[len(str)-1]
		}
	}
	if imageTag != desiredVersion {
		return false, nil
	}
	return true, nil
}

func (o *operator) IsRunningDesiredVersion(ctx context.Context) (bool, error) {
	// Get the desired Version
	image := o.env.AROOperatorImage()
	desiredVersion := "latest"
	if strings.Contains(image, ":") {
		str := strings.Split(image, ":")
		desiredVersion = str[len(str)-1]
	}
	if o.oc.Properties.OperatorVersion != "" {
		desiredVersion = o.oc.Properties.OperatorVersion
	}

	// Check if aro-operator-master is running desired version
	ok, err := checkOperatorDeploymentVersion(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-master", desiredVersion)
	if !ok || err != nil {
		return ok, err
	}
	ok, err = checkPodImageVersion(ctx, o.kubernetescli.CoreV1().Pods(pkgoperator.Namespace), "aro-operator-master", desiredVersion)
	if !ok || err != nil {
		return ok, err
	}
	// Check if aro-operator-worker is running desired version
	ok, err = checkOperatorDeploymentVersion(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-worker", desiredVersion)
	if !ok || err != nil {
		return ok, err
	}
	ok, err = checkPodImageVersion(ctx, o.kubernetescli.CoreV1().Pods(pkgoperator.Namespace), "aro-operator-worker", desiredVersion)
	if !ok || err != nil {
		return ok, err
	}
	return true, nil
}

func checkIngressIP(ingressProfiles []api.IngressProfile) (string, error) {
	if ingressProfiles == nil || len(ingressProfiles) < 1 {
		return "", errors.New("no Ingress Profiles found")
	}
	ingressIP := ingressProfiles[0].IP
	if len(ingressProfiles) > 1 {
		for _, p := range ingressProfiles {
			if p.Name == "default" {
				return p.IP, nil
			}
		}
	}
	return ingressIP, nil
}

func isCRDEstablished(crd *extensionsv1.CustomResourceDefinition) bool {
	m := make(map[extensionsv1.CustomResourceDefinitionConditionType]extensionsv1.ConditionStatus, len(crd.Status.Conditions))
	for _, cond := range crd.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[extensionsv1.Established] == extensionsv1.ConditionTrue &&
		m[extensionsv1.NamesAccepted] == extensionsv1.ConditionTrue
}
