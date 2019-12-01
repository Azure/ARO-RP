# Upstream differences

This file catalogues the differences of install approach between ARO and
upstream OCP.

## Installer carry patches

* CARRY: HACK: remove dependency on github.com/openshift/installer/pkg/terraform
  from pkg/asset/cluster

  This enables dep to import the installer as a library as well as significantly
  decreasing compilation time and binary footprint

* CARRY: PARTIAL: allow platform credentials to be prepopulated in
  installconfig.PlatformCreds

  This avoids the installer going to disk to fetch the credentials, enabling one
  installer to handle multiple cluster installations simultaneously

* CARRY: allow end user to specify Azure resource group on cluster creation

  This allows the RP to specify the cluster's resource group.  TODO: reduce the
  scope of this patch to get this upstream; don't allow end-users to choose
  their cluster resource group

* CARRY: HACK: don't use managed identity on ARO

  At the moment OCP on Azure uses MSI for kubelets and controllers and one or
  more service principals for operators.  For now on ARO, simplify to all
  components using the user-provided SP.  Later, we'll reinstate a separate
  managed identity at least for worker kubelets

## Installation differences

* ARO persists the install graph in the cluster storage account in a new "aro"
  container / "graph" blob

* No managed identity (for now)

* installconfig.ClusterID.InfraID is hard-coded to "aro"

* API server public IP domain name label is an 8 character random label, not the
  infraID
