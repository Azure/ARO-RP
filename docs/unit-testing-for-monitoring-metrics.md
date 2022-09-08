
# Testing ARO Monitor Metrics



## The Monitor Architecture

The ARO monitor component (the part of the aro binary you activate when you execute ./cmd/aro monitor) collects and emits the various metrics about cluster health, the monitor's own health, and e2e tests we want to see in Geneva.


![Aro Monitor Architecture](img/AROMonitor.png "Aro Monitor Architecture")

To send data to Geneva the monitor uses an instance of a Geneva MDM container as a proxy of the Geneva API. The MDM container accepts statsd formatted data (the Azure Geneva version of statsd, that is) over a UNIX (Domain) socket. The MDM container then forwards the metric data over a https link to the Geneva API. Please note that a Unix socket can only be accessed from the same machine.

The monitor picks the required information about which clusters should actually monitor from its corresponding Cosmos DB. If multiple monitor instances run in parallel  (i.e. connect to the same database instance) as is the case in production, they negotiate which instance monitors what cluster (see : [monitoring.md](./monitoring.md)).


# Unit Testing Setup

If you work on monitor metrics in local dev mode (RP_MODE=Development) you most likely want to see your data somewhere in Geneva INT (https://jarvis-west-int.cloudapp.net/) before you ship your code.

There are two ways to set to achieve this:
- Run the Geneva MDM container locally
- Spawn a VM, start the Geneva container there and connect/tunnel to it.

and two protocols to chose from:
- Unix Domain Sockets, which is the way production is currently (April 2022) run
- or UDP, which is much easier to use and is the way it will be used on kubernetes clusters in the future

## Common Setup

No matter which of the two methods of running the Geneva MDM container you choose (locally or on a remote VM), make sure you:
- `az login`  into your subscription
- run `SECRET_SA_ACCOUNT_NAME=rharosecretsdev make secrets` to pull the latest env variables, certs, and keys
- run `source ./env` to set your general RP environment variables
- run `source ./secrets/mdm_env` to set your MDM related environment variables

  If the `./secrets/mdm_env` file does not exist, then create it by running:

    ```bash
  cat >secrets/mdm_env <<EOF
  export CLUSTER_MDM_ACCOUNT='AzureRedHatOpenShiftCluster'
  export CLUSTER_MDM_NAMESPACE='<PUT YOUR CLUSTER NAMESPACE HERE>'
  export HOSTNAME=$( hostname )
  export MDM_IMAGE='linuxgeneva-microsoft.azurecr.io/genevamdm:master_20221018.2'
  export MDM_E2E_ACCOUNT='AzureRedHatOpenShiftE2E'
  export MDM_E2E_NAMESPACE='E2E'
  export MDM_FRONTEND_URL='https://int2.int.microsoftmetrics.com/'
  export MDM_SOURCE_ENVIRONMENT="$LOCATION"
  export MDM_SOURCE_ROLE='rp'
  export MDM_SOURCE_ROLE_INSTANCE="$HOSTNAME"
  export MDM_VM_NAME="$RESOURCEGROUP-mdm"
  export MDM_VM_PRIVATE='true'
  EOF
  ```

  Change the `CLUSTER_MDM_ACCOUNT` and `CLUSTER_MDM_NAMESPACE` values as necessary
- have the correct `./secrets/rp-metrics-int.pem` certificate. If the certificate is not correct or does not exist, then it can be obtained from the `svc` keyvault in public int
  
If any changes were made inside the `./secrets` folder, then please run `make secrets-update` to upload it to your storage account so other people on your team can access it via `make secrets`.

## Method Specific Setup

### Local Setup

There are no additional tasks for setting up a local testing environment, the `Common Setup` tasks cover them all

#### Starting the monitor

Use `go run -tags aro ./cmd/aro monitor`  to start the monitor. You want to check what the current directory of your monitor is, because that's the folder the monitor will use to search for the mdm_statds.socket file and that needs to match where your mdm container or the socat command creates it. Please note that in local dev mode the monitor will silently ignore if it can't connect to the socket.

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
    "env": {"CLUSTER_MDM_ACCOUNT": "<PUT YOUR CLUSTER ACCOUNT HERE>",
    "CLUSTER_MDM_NAMESPACE":"<PUT YOUR CLUSTER NAMESPACE HERE>",
    "MDM_E2E_ACCOUNT"="<PUT YOUR E2E ACCOUNT HERE>"
    "MDM_E2E_NAMESPACE"="<PUT YOUR E2E NAMESPACE HERE>"
    }    
},
````

### Remote Container Setup

If you can't run the container locally (because you run on macOS and your container tooling does not support Unix Sockets, which is true both for Docker for Desktop or podman) and or don't want to, you can bring up the container on a Linux VM and connect via a socat/ssh chain:
![alt text](img/SOCATConnection.png "SOCAT chain")

The [deploy script](../hack/local-monitor-testing/deploy_MDM_VM.sh) deploys such a VM called $RESOURCEGROUP-mdm on Azure, configures it and installs the mdm container.

The [start network script](../hack/local-monitor-testing/startMDMNetwork.sh) can then be used to established the network connection as depicted in the diagram.

Each script will use the `./secrets/mdm_id_rsa` ssh key to connect to the VM. If this key does not exist, a new one can be generated by running
```bash
ssh-keygen -f secrets/mdm_id_rsa -N ''
```
`make secrets-update` can then be used to upload the key to your storage account so other people on your team can access it via `make secrets`.

The network script will effectively start run three commands (with more error handling):

```bash
MDM_VM_IP=<Either public or private IP of your VM, depending on whether $MDM_VM_PRIVATE is set or not>

BASE=$( git rev-parse --show-toplevel)
SOCKETFILE=$BASE/cmd/aro/mdm_statsd.socket

ssh -i ./secrets/mdm_id_rsa cloud-user@$MDM_VM_IP 'sudo socat -v TCP-LISTEN:12345,fork UNIX-CONNECT:/var/etw/mdm_statsd.socket'

ssh -i ./secrets/mdm_id_rsa cloud-user@$MDM_VM_IP -N -L 12345:127.0.0.1:12345

socat -v UNIX-LISTEN:$SOCKETFILE,fork TCP-CONNECT:127.0.0.1:12345
```

For debugging it might be useful to run these commands manually in three different terminals to see where the connection might break down. The docker log file should show if data flows through or not, too.

If this VM is not going to have a public IP address, and only be available over VPN, then please ensure you are connected the the appropriate VPN before running either script.

#### Stopping the Network script

Stop the script with Ctrl-C. The script then will do its best to stop the ssh and socal processes it spawned.

## Injecting Test Data into Geneva INT

Once your monitor code is done you will want to create pre-aggregates, dashboards and alert on the Geneva side and test with a variety of data.
Your end-2-end testing with real cluster will generate some data and cover many test scenarios, but if that's not feasible or too time-consuming you can inject data directly into the Genava mdm container via the socat/ssh network chain.

An example metric script is shown below. 

````
myscript.sh | socat TCP-CONNECT:127.0.0.1:12345 - 
````
or 
````
myscript.sh | socat UNIX-CONNECT:$SOCKETFILE - 
````
(see above of the $SOCKETFILE )


### Sample metric script

````
#!/bin/bash
# example metric 
CLUSTER="< your testcluster example name>"
SUBSCRIPTION="<  your subscription here >"
METRIC="< your metric name>"
ACCOUNT="< your CLUSTER_MDM_ACCOUNT >"
NAMESPACE="< CLUSTER_MDM_NAMESPACE>"
DIM_HOSTNAME="<your hostname >"
DIM_LOCATION="< your region >"
DIM_NAME="pod-$CLUSTER"
DIM_NAMESPACE="< somenamespace> "
DIM_RESOURCEGROUP="< your resourcegroup> "
DIM_RESOURCEID="/subscriptions/$SUBSCRIPTION/resourceGroups/$DIM_RESOURCEGROUP/providers/Microsoft.RedHatOpenShift/openShiftClusters/$CLUSTER"
DIM_RESOURCENAME=$CLUSTER

### or read data from file, like: data=$( cat mydatafile )
data="10 11 12 13 13 13 13 15 16 19 20 21 25"
SLEEPTIME=60
for MET in $data ;do
DATESTRING=$( date -u +'%Y-%m-%dT%H:%M:%S.%3N' )
OUT=$( cat << EOF 
{"Metric":"$METRIC",
"Account":"$ACCOUNT",
"Namespace":"$NAMESPACE",
"Dims":
    {"hostname":"$DIM_HOSTNAME",
    "location":"$DIM_LOCATION",
    "name":"$DIM_NAME",
    "namespace":"$DIM_NAMESPACE",
    "resourceGroup":"$DIM_RESOURCEGROUP",
    "resourceId":"$DIM_RESOURCEID",
    "resourceName":"$DIM_RESOURCENAME",
    "subscriptionId":"$SUBSCRIPTION"
},
"TS":"$DATESTRING"}:${MET}|g
EOF
)

echo $OUT
sleep $SLEEPTIME
done

````

## Injecting E2E Test Metric Data into Geneva INT

With the the mdm container running, either locally or remote with a socat link, running `make test-e2e` with the `MDM_E2E_ACCOUNT` and `MDM_E2E_NAMESPACE` environment variables set will send metric data to Geneva.

## Finding your data

If all goes well, you should see your cluster metric data in the Jarvis metrics list (Geneva INT (https://jarvis-west-int.cloudapp.net/) -> Manage ->  Metrics) under the account and namespace you specified in `CLUSTER_MDM_ACCOUNT` and `CLUSTER_MDM_NAMESPACE` and also be available is the dashboard settings. E2E metric data will be present in the account and namespace you specified with `MDM_E2E_ACCOUNT` and `MDM_E2E_NAMESPACE`.
