# Upstream differences

This file catalogues the differences of install approach between ARO and
upstream OCP.

## Installer carry patches

See
https://github.com/jim-minter/installer/compare/release-4.5...jim-minter:release-4.5-azure .

## Installation differences

* ARO persists the install graph in the cluster storage account in a new "aro"
  container / "graph" blob.

* No managed identity (for now).

* No IPv6 support (for now).

* Upstream installer closely binds the installConfig (cluster) name, cluster
  domain name, infra ID and Azure resource name prefix.  ARO separates these out
  a little.  The installConfig (cluster) name and the domain name remain bound;
  the infra ID and Azure resource name prefix are taken from the ARO resource
  name.

* API server public IP domain name label is not set.

* ARO uses first party RHCOS OS images published by Microsoft.

* ARO never creates xxxxx-bootstrap-pip-* for bootstrap VM, or the corresponding
  NSG rule.

* ARO does not create a outbound-provider Service on port 27627.

* ARO deploys a private link service in order for the RP to be able to
  communicate with the cluster.
