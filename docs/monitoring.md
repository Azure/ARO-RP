# Monitoring

## Initial goals

* Jump start our monitoring capabilities to get basic visibility quickly and
  enable rapid iteration.
* Enable capabilities which cluster admins cannot significantly tamper with.
* Straightforward operational prerequisites (reliability, upgradeability,
  observability, basic scalability, state management, management in multiple
  regions, etc.)

The first two imply external monitoring, but not to the exclusion of adding
monitoring from inside the cluster as well as a complementary near-term goal.

## Implementation

* Monitoring is horizontally scalable, active/active.
* Every monitor process advertises its liveness to the database by updating its
  own MonitorDocument named with its UUID.  These MonitorDocuments have a ttl
  set, so each will disappear from the database if it is not regularly
  refreshed.
* Every monitor process competes for a lease on a MonitorDocument called
  "master".  The master lease owner lists the advertised monitors (hopefully
  including itself) and shares ownership of 256 monitoring buckets evenly across
  the monitors.
* Every monitor process regularly checks the "master" MonitorDocument to learn
  what buckets it has been assigned.
* Every cluster is placed at create time into one of the 256 buckets using a
  uniform random distribution.
* Each monitor uses a Cosmos DB change feed to keep track of database state
  locally (like k8s list/watch).  At startup, the cosmos DB change feed returns
  the current state of all of the OpenShiftClusterDocuments; subsequently as
  OpenShiftClusterDocuments it returns the updated documents.
* At the moment of writing, the change feed does not log record deletions. It logs
  only changes. Deallocated clusters are deleted from the monitoring list only if
  they were seen in the `DeletingProvisioningState` by the monitor.
  The monitor reads the change feed every 10 seconds, so we should avoid
  cases when `OpenShiftClusterDocuments` have the `DeletingProvisioningState` for 
  less than 10 seconds.
* Each monitor aims to check each cluster it "owns" every 5 minutes; it walks
  the local database map and distributes checking over lots of local goroutine
  workers.
* Monitoring stats are output to mdm via statsd.

## Back-of-envelope calculations

* To support 50,000 clusters/RP with (say) 3 monitors, and check every cluster
  every 5 minutes, each monitor will need to retire 55 checks per second.
* If each check is allowed up to 30 seconds to run, that implies 1650 active
  goroutines per monitor.
* If each cluster's cached data model takes 2KB and each goroutine takes 2KB,
  memory usage per monitor would be around 103MB.

## Running monitor locally and capturing metrics

### Pre-requisites

- [Prepare Your Development Environment](https://github.com/Azure/ARO-RP/blob/master/docs/prepare-your-dev-environment.md).
- Start the local RP and create a cluster
- Stop the local RP

### Capturing the metrics from the monitor

Run the following commands (install `socat` in your system if it's not present)

```
socat -v UNIX-LISTEN:mdm_statsd.socket,fork STDOUT
```

In another terminal:

```
make runlocal-monitor
````

Monitor will start sending metrics about itself and any cluster that has been created
by the local RP.

`socat` will start displaying the raw statsd packets, containing metric name, labels
and values for each metric gathered by monitor. If you are interested into specifics,
you may need to grep by the metric name or string that you're looking for.
