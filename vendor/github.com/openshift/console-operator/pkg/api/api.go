package api

const (
	TargetNamespace    = "openshift-console"
	ConfigResourceName = "cluster"
)

// consts to maintain existing names of various sub-resources
const (
	ClusterOperatorName                 = "console"
	OpenShiftConsoleName                = "console"
	OpenShiftConsoleNamespace           = TargetNamespace
	OpenShiftConsoleOperatorNamespace   = "openshift-console-operator"
	OpenShiftConsoleOperator            = "console-operator"
	OpenShiftConsoleConfigMapName       = "console-config"
	OpenShiftConsolePublicConfigMapName = "console-public"
	ServiceCAConfigMapName              = "service-ca"
	OpenShiftConsoleDeploymentName      = OpenShiftConsoleName
	OpenShiftConsoleServiceName         = OpenShiftConsoleName
	OpenShiftConsoleRouteName           = OpenShiftConsoleName
	OpenShiftConsoleDownloadsRouteName  = "downloads"
	OAuthClientName                     = OpenShiftConsoleName
	OpenShiftConfigManagedNamespace     = "openshift-config-managed"
	OpenShiftConfigNamespace            = "openshift-config"
	OpenShiftCustomLogoConfigMapName    = "custom-logo"
	TrustedCAConfigMapName              = "trusted-ca-bundle"
	TrustedCABundleKey                  = "ca-bundle.crt"
	TrustedCABundleMountDir             = "/etc/pki/ca-trust/extracted/pem"
	TrustedCABundleMountFile            = "tls-ca-bundle.pem"
	OCCLIDownloadsCustomResourceName    = "oc-cli-downloads"
	ODOCLIDownloadsCustomResourceName   = "odo-cli-downloads"
)
