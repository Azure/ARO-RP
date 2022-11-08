package api

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	TargetNamespace    = "openshift-console"
	ConfigResourceName = "cluster"
)

// consts to maintain existing names of various sub-resources
const (
	ClusterOperatorName                       = "console"
	OpenShiftConsoleName                      = "console"
	OpenShiftConsoleNamespace                 = TargetNamespace
	OpenShiftConsoleOperatorNamespace         = "openshift-console-operator"
	OpenShiftConsoleOperator                  = "console-operator"
	OpenShiftConsoleConfigMapName             = "console-config"
	OpenShiftConsolePublicConfigMapName       = "console-public"
	ServiceCAConfigMapName                    = "service-ca"
	DefaultIngressCertConfigMapName           = "default-ingress-cert"
	OAuthServingCertConfigMapName             = "oauth-serving-cert"
	OAuthConfigMapName                        = "oauth-openshift"
	OpenShiftConsoleDeploymentName            = OpenShiftConsoleName
	OpenShiftConsoleServiceName               = OpenShiftConsoleName
	OpenShiftConsolePDBName                   = OpenShiftConsoleName
	OpenshiftConsoleRedirectServiceName       = "console-redirect"
	OpenShiftConsoleRouteName                 = OpenShiftConsoleName
	OpenshiftConsoleCustomRouteName           = "console-custom"
	DownloadsResourceName                     = "downloads"
	OpenShiftConsoleDownloadsRouteName        = DownloadsResourceName
	OpenShiftConsoleDownloadsDeploymentName   = DownloadsResourceName
	OpenShiftConsoleDownloadsPDBName          = DownloadsResourceName
	OAuthClientName                           = OpenShiftConsoleName
	OpenShiftConfigManagedNamespace           = "openshift-config-managed"
	OpenShiftConfigNamespace                  = "openshift-config"
	OpenShiftCustomLogoConfigMapName          = "custom-logo"
	TrustedCAConfigMapName                    = "trusted-ca-bundle"
	TrustedCABundleKey                        = "ca-bundle.crt"
	TrustedCABundleMountDir                   = "/etc/pki/ca-trust/extracted/pem"
	TrustedCABundleMountFile                  = "tls-ca-bundle.pem"
	OCCLIDownloadsCustomResourceName          = "oc-cli-downloads"
	ODOCLIDownloadsCustomResourceName         = "odo-cli-downloads"
	HubClusterName                            = "local-cluster"
	ManagedClusterLabel                       = "managed-cluster"
	ManagedClusterConfigMapName               = "managed-clusters"
	ManagedClusterConfigMountDir              = "/var/managed-cluster-config"
	ManagedClusterConfigKey                   = "managed-clusters.yaml"
	ManagedClusterAPIServerCertMountDir       = "/var/managed-cluster-api-server-certs"
	ManagedClusterAPIServerCertName           = "managed-cluster-api-server-cert"
	ManagedClusterAPIServerCertKey            = "ca-bundle.crt"
	ManagedClusterOAuthServerCertMountDir     = "/var/managed-cluster-oauth-server-certs"
	ManagedClusterOAuthServerCertName         = "managed-cluster-oauth-server-cert"
	ManagedClusterOAuthServerCertKey          = "ca-bundle.crt"
	ManagedClusterOAuthClientName             = "console-managed-cluster-oauth-client"
	OAuthClientManagedClusterViewName         = "console-oauth-client"
	CreateOAuthClientManagedClusterActionName = "console-create-oauth-client"
	OAuthServerCertManagedClusterViewName     = "console-oauth-server-cert"

	PluginI18nAnnotation = "console.openshift.io/use-i18n"

	ManagedClusterViewAPIGroup     = "view.open-cluster-management.io"
	ManagedClusterViewAPIVersion   = "v1beta1"
	ManagedClusterViewResource     = "managedclusterviews"
	ManagedClusterActionAPIGroup   = "action.open-cluster-management.io"
	ManagedClusterActionAPIVersion = "v1beta1"
	ManagedClusterActionResource   = "managedclusteractions"

	ConsoleContainerPortName    = "https"
	ConsoleContainerPort        = 443
	ConsoleContainerTargetPort  = 8443
	RedirectContainerPortName   = "custom-route-redirect"
	RedirectContainerPort       = 8444
	RedirectContainerTargetPort = RedirectContainerPort
	ConsoleServingCertName      = "console-serving-cert"
	DownloadsPortName           = "http"
	DownloadsPort               = 8080
)

var (
	ManagedClusterViewGroupVersionResource = schema.GroupVersionResource{
		Group:    ManagedClusterViewAPIGroup,
		Version:  ManagedClusterViewAPIVersion,
		Resource: ManagedClusterViewResource,
	}
	ManagedClusterActionGroupVersionResource = schema.GroupVersionResource{
		Group:    ManagedClusterActionAPIGroup,
		Version:  ManagedClusterActionAPIVersion,
		Resource: ManagedClusterActionResource,
	}
)
