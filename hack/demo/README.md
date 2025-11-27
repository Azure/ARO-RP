# Cluster failover testing

These scripts simplify the process of demonstrating the [business continuity and disaster recovery](https://learn.microsoft.com/en-us/azure/cloud-adoption-framework/scenarios/app-platform/azure-red-hat-openshift/operations#business-continuity-and-disaster-recovery-bcdr) capabilities we document for customers

# Overview

The summary for the BCDR plan is to have multiple clusters behind Azure Front Door (or some other form of load balancing) and if/when a cluster becomes unavailable for whatever reason (eg: a datacenter outage, etc), traffic gets routed to the still-healthy cluster(s)

We will simulate this by creating two clusters in two different regions, each hosting a static web server that will identify the cluster when queried, then placing the static web servers behind Azure Front Door. AFD will be configured to route all traffic to one web server until it fails its health checks, in which case the second web server will receive all traffic. We will then simulate an outage and verify that traffic is routed to the second web server

# Demo setup steps

1. Log into Azure
```
az login
```

2. Run the cluster 1 setup script from its directory. This will create a cluster with a static web server identifying the cluster
```
cd ./script1
./staticweb.sh
```

3. Run the cluster 2 setup script from its directory. This will create another cluster with a static web server identifying the cluster
```
cd ../script2
./staticweb.sh
```

4. Run the setup script for the front door configuration. This will create an Azure Front Door config prioritizing all traffic to the web server on cluster 1 with the web server on cluster 2 as a backup.

> [!IMPORTANT]
> Tt time of writing, AFD configs take around an hour to sync and become active!

```
cd ..
./frontdoor.sh
```

# Demo testing and documenting

We need to document the states of the resources before the simulated incident, during the simulated incident and after recovery (with datestamps for each)

## Testing steps

1. Go into the Azure portal, find the bcdrfrontdoor resource group, screenshot the Azure Front Door route and origin group and save them as AFD\_route.png and AFD\_origin\_group.png


2. Display the routes for the static web servers and their outputs (the frontdoor script will already output its URL)
```
cat cluster1/cluster1_route.txt
curl $(cat cluster1/cluster1_route.txt)
cat cluster2/cluster2_route.txt
curl $(cat cluster2/cluster2_route.txt)
curl <frontdoor URL>
date
```

Screenshot the above and save it into a file: before\_curl\_routes.png

3. Simulate an incident for cluster1
```
KUBECONFIG=cluster1/cluster1.kubeconfig oc scale deploy -n staticweb --replicas 0 nginx
date
```

Screenshot the above and save it into a file: incident\_creation.png

4. Verify that traffic to the front door gets routed to the webserver on the failover cluster.

> [!NOTE]
> It will take a few minutes for Azure Front Door to mark the webserver on the main cluster as unhealthy and begin routing traffic to the webserver on the failover cluster

```
curl <frontdoor URL>
date
```

Screenshot the above (once the frontdoor URL shows output from the webserver on the failover cluster) and save it as incident\_verification.png

5. End the simulated incident and verify that traffic returns to the webserver on the main cluster

> [!NOTE]
> It will take a few minutes for Azure Front Door to mark the webserver on the main cluster as healthy and begin routing traffic to it again

```
KUBECONFIG=cluster1/cluster1.kubeconfig oc scale deploy -n staticweb --replicas 2 nginx
sleep 300
curl <frontdoor url>
date
```

Screenshot the above and save it to a file: post\_recovery.png
