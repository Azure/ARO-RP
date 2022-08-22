package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

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
	"github.com/Azure/ARO-RP/pkg/env"
	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/genevalogging"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/ready"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

//go:embed staticresources
var embeddedFiles embed.FS

type Operator interface {
	CreateOrUpdate(context.Context) error
	IsReady(context.Context) (bool, error)
	IsRunningDesiredVersion(context.Context) error
}

type operator struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	arocli        aroclient.Interface
	extensionscli extensionsclient.Interface
	kubernetescli kubernetes.Interface
	dh            dynamichelper.Interface
}

func New(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, arocli aroclient.Interface, extensionscli extensionsclient.Interface, kubernetescli kubernetes.Interface) (Operator, error) {
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

		arocli:        arocli,
		extensionscli: extensionscli,
		kubernetescli: kubernetescli,
		dh:            dh,
	}, nil
}

type deploymentData struct {
	Image              string
	Version            string
	IsLocalDevelopment bool
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
	templatesArray := []*template.Template{templatesMaster, templatesRoot, templatesWorker}

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

func (o *operator) createDeploymentData() deploymentData {
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

	return deploymentData{
		IsLocalDevelopment: o.env.IsLocalDevelopmentMode(),
		Image:              image,
		Version:            version,
	}
}

func (o *operator) createObjects() ([]kruntime.Object, error) {
	deploymentData := o.createDeploymentData()
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

func (o *operator) resources() ([]kruntime.Object, error) {
	// first static resources from Assets

	results, err := o.createObjects()
	if err != nil {
		return nil, err
	}

	// then dynamic resources
	key, cert := o.env.ClusterGenevaLoggingSecret()
	gcsKeyBytes, err := utiltls.PrivateKeyAsBytes(key)
	if err != nil {
		return nil, err
	}

	gcsCertBytes, err := utiltls.CertAsBytes(cert)
	if err != nil {
		return nil, err
	}

	ps, err := pullsecret.Build(o.oc, "")
	if err != nil {
		return nil, err
	}

	vnetID, _, err := subnet.Split(o.oc.Properties.MasterProfile.SubnetID)
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
		cluster,
	), nil
}

func (o *operator) CreateOrUpdate(ctx context.Context) error {
	resources, err := o.resources()
	if err != nil {
		return err
	}

	err = dynamichelper.Prepare(resources)
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

func (o *operator) IsReady(ctx context.Context) (bool, error) {
	ok, err := ready.CheckDeploymentIsReady(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-master")()
	if !ok || err != nil {
		return ok, err
	}
	ok, err = ready.CheckDeploymentIsReady(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-worker")()
	if !ok || err != nil {
		return ok, err
	}

	return true, nil
}

func checkOperatorDeploymentVersion(ctx context.Context, cli appsv1client.DeploymentInterface, name string, gitCommit string) error {
	d, err := cli.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if d.Labels["version"] != gitCommit {
		return errors.New(name + " is not running the desired version: " + gitCommit)
	}

	return nil
}

func checkPodImageVersion(ctx context.Context, cli corev1client.PodInterface, namespace string, gitCommit string) error {
	podList, err := cli.List(ctx, metav1.ListOptions{LabelSelector: "app=" + namespace})
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		imageTag := strings.Split(pod.Spec.Containers[0].Image, ":")
		if imageTag[len(imageTag)-1] != gitCommit {
			return errors.New(pod.Name + " pod of namespace " + pod.Namespace + " is not running the desired version: " + gitCommit)
		}
	}
	return nil
}

func (o *operator) IsRunningDesiredVersion(ctx context.Context) error {
	// check if aro-operator-master is running desired version
	err := checkOperatorDeploymentVersion(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-master", version.GitCommit)
	if err != nil {
		return err
	}
	err = checkPodImageVersion(ctx, o.kubernetescli.CoreV1().Pods(pkgoperator.Namespace), "aro-operator-master", version.GitCommit)
	if err != nil {
		return err
	}
	// check if aro-operator-worker is running desired version
	err = checkOperatorDeploymentVersion(ctx, o.kubernetescli.AppsV1().Deployments(pkgoperator.Namespace), "aro-operator-worker", version.GitCommit)
	if err != nil {
		return err
	}
	err = checkPodImageVersion(ctx, o.kubernetescli.CoreV1().Pods(pkgoperator.Namespace), "aro-operator-worker", version.GitCommit)
	if err != nil {
		return err
	}
	return nil
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
