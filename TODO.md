## TODO

* Unit tests
  * MissingFields
* E2E tests
* Subscription lifecycle API (need clarity from Microsoft)
* RBAC? (need clarity from Microsoft)
* Metrics
* Admin API
* Signed cluster TLS certificates
* BBM
* Carefully sort out URL case sensitivity
* What to do with storage account at end of deploy?
* Check about route table
* Deploy to named resource group
* Sort out mess of clients
* Check insights, telemeter settings on deployed cluster
* Was SNAT issue on internal LB solved?
* Design and implement how we will configure release-payload
    * Release payload hosting in Azure
    * Offline cluster
    * "Hot patch" path in production

(Lower priority)

* Swagger example generation
* Implement ARM move API
* Implement paging on list APIs
* Formal ARM asynchronous operation (Azure-AsyncOperation header)
* Make installer providers pluggable
* Move to go modules