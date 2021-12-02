.. :changelog:

Release History
===============

0.1.0
++++++
* Initial release.

0.2.0
++++++
* Use API paging.
* Remove azext.maxCliCoreVersion.

0.3.0
++++++
* Add --pull-secret argument to `az aro create`.
* Stop advertising Python 2.7, 3.5 support.
* Migrate to GA API.

0.4.0
++++++
* Default worker VM size to Standard_D4s_v3.

1.0.0
++++++
* Remove preview flag.

1.0.1
++++++
* Switch to new preview API

1.0.2
++++++
* Add support for list admin credentials (getting kubeconfig)

1.0.3
++++++
* Fix role assignment bug

1.0.4
++++++
* Remove unused code (identifier URLs)

1.0.5
++++++
* Fixed get-admin-kubeconfig Enums for Feature state no longer available in AzureCLI
* Added saving kubeconfig to file

1.0.6
++++++
* Fixed backwards compatibility with Python3.6
