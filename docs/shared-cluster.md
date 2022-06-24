# Shared Cluster

The shared cluster now lives in the AME tenant. We have access to credentials to use the cluster, but any "operational" action must go through JIT and the Azure Portal. 

## Overview

The following diagram is the overview of where our shared cluster lives, and how we access it.

* Here is a link to the living lucid chart diagram: [here](https://lucid.app/lucidchart/1e415fe2-af56-4409-abc6-3bdf96f1bffd/edit?beaconFlowId=AB8BF83B17AD4D23&invitationId=inv_a6e62e97-2bcf-4c3a-b7b0-5dada5ea075d&page=0_0#)

## Diagram

![alt text](img/sharedcluster.png "Shared Cluster Overview")



## Authentication

We have the kubeadmin credentials as well as the kubeadmin kubeconfig file. You can use either to authenticate to the cluster.

* Make secrets:

```
SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets
```

* Oc login, assuming `env` sources `secrets/env`:
```
. ./env
make shared-cluster-login
```

* Use kubeconfig
```
export KUBECONFIG=$PWD/secrets/shared-cluster.kubeconfig
```

## Creating / Deleting the Shared Cluster

The shared cluster has the following attributes:

* Tenant: AME

* Subscription: ARO CI/E2E

* Region: Westcentralus

* Name: shared-cluster

* Resource group name: shared-cluster

* Cluster resource group name: aro-shared-cluster


### Create / Delete
To create/ delete/ administer the cluster from az cli you must have proper permissions (JIT in the case of AME).

* Create:

```bash
./hack/shared-cluster.sh create
```

* Delete:

```bash
./hack/shared-cluster.sh delete
```
