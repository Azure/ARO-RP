// pkg/util/namespace/monitored.go
package namespace

// MonitoredNamespaces is the curated set of OpenShift/ARO-managed namespaces
// for which the cluster monitor emits pod.conditions metrics.
// This is intentionally smaller than IsOpenShiftNamespace to minimize API load.
var MonitoredNamespaces = []string{
	"openshift-apiserver",
	"openshift-azure-logging",
	"openshift-azure-operator",
	"openshift-etcd",
	"openshift-ingress",
	"openshift-kube-apiserver",
	"openshift-machine-config-operator",
	"openshift-machine-api",
	"openshift-monitoring",
}
