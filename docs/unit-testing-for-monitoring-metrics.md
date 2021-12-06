
# Testing ARO Monitor Metrics



## The Monitor Architecture

The ARO monitor component (the part of the aro binary you activate when you execute ./cmd/aro monitor) collects and emits the various metrics about cluster health (and its own) we want to see in Geneva. 

To send data to Geneva the monitor uses an instance of a Geneva MDM container as a proxy of the Geneva API. The MDM container accepts statsd formatted data (the Azure Geneva version of statsd, that is) over a UNIX (Domain) socket. The MDM container then forwards the metric data over a https link to the Geneva API. Please note that using a Unix socket can only be accessed from then same machine. 

The monitor picks the required information about which clusters should actualyl monitor from its corresponding Cosmos DB. If multiple monitor instances run in parallel  (i.e. connect to the same database instance) as is the case in production, they negotiate which instance monitors what cluster (see : [monitoring.md](./monitoring.md)). 


![Aro Monitor Architecture](img/AROMonitor.png "Aro Monitor Architecture")


## Unit Testing Setup

There are two ways set up: 
- Run the Geneva container locally.
- Spawn a VM, start the Geneva container there and connect/tunnel to it.

### Local container setup

An example docker command to start the container locally is here (you will need to adapt some parameters):
[Example](../hack/local-monitor-testing/sample/dockerStartCommand.sh)

Two things to adapt:
* Amongst other things container needs to be provided with the Geneva key and certificate. For the INT instance that is the rp-metrics-int.pem you find in the secrets folder after running `make secrets`.  Copy that to /etc/mdm.pem or adapt the volume mount accordingly. The mdm container logs will tell you of that worked or not.
* When you start the montitor locally in local dev mode, the monitor looks for the Unix Socket file mdm_statsd.socket in the current path (usually ./cmd/aro folder) . Adapt the path in the start command accordingly.

### Remote container setup

If you can't run the container locally (because you run on macOS and you container tooling does not support Unix Sockets, which is true both for Docker for Desktop or podman) and or don't want to, you can bring up the container on a Linux VM and connect via a socat/ssh chain:
![alt text](img/SOCATConnection.png "SOCAT chain")

The [deploy script](../hack/local-monitor-testing/deploy_MDM_VM.sh) deploys such a VM on Azure (if you ./env things properly), configures it and installs the container.

The [start script](../hack/local-monitor-testing/startMDMNetwork.sh)  can then be used to established the network connection as depicted in the diagram. For local VMs you may want to skip the ssh tunnel step.


### Starting the monitor

When starting the monitor , make sure to have your

- CLUSTER_MDM_ACCOUNT
- CLUSTER_MDM_NAMESPACE
  
environment variables set to Geneva account and namespace where you metrics is supposed to land in Geneva INT (https://jarvis-west-int.cloudapp.net/)

Use `go run -tags aro ./cmd/aro monitor`  to start the monitor. You want to check what the current directory of your monitor is, because that's the folder the monitor will use to search for the mdm_statds.socket file, which needs to match where your mdm container or the socat command creates it.

A VS Code launch config that does the same would look like.

````
{
            "name": "Launch Monitor",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./cmd/aro",
            "buildFlags": "-tags aro",
            "console": "integratedTerminal",
            "args": ["-loglevel=debug",
                "monitor",
            ],    
            "env": {"CLUSTER_MDM_ACCOUNT": "<PUT YOUR NAMESPACE HERE>",
            "CLUSTER_MDM_NAMESPACE":"<PUT YOUR NAMESPACE HERE>" }    
        },
````