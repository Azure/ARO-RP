# Azure Red Hat OpenShift CLI preview extension

ARO CLI extension code

## Configure & generate extensions

All these steps assumes you read throuth documents in useful links sections
and you are already familiar with terminology

1. Clone required dev repositories:
```
git clone https://github.com/Azure/azure-cli
git clone https://github.com/Azure/azure-cli-extensions
```

1. Setup virtual env in the repository which will unify all projects,
including `rp`. I do this on `$GOPATH/src/github.com` level.
```
python3 -m venv env
source env/bin/activate
```

1. Install pre-requisited
```
pip install azdev
azdev setup
```

1. Add external extension repistory
```
azdev extension repo add /home/mjudeiki/go/src/github.com/mjudeikis/azure-cli-aro
```

From this point, if you want develop existing extension - add extension to az:
client. Otherwise generate new from the template
```
azdev extension add aro-preview
```

### Generate extension template

1. Generate extension template
```
azdev extension create aro-preview --client-name AzureRedHatOpenShiftClient --operation-name OpenShiftClustersOperations --local-sdk /home/mjudeiki/go/src/github.com/jim-minter/rp/az/azure-python-sdk/2019-12-31-preview/redhatopenshift/ --sdk-property=resource_name --github-alias=mjudeikis --repo-name /home/mjudeiki/go/src/github.com/mjudeikis/azure-cli-aro
```

## Useful links

* https://github.com/Azure/azure-cli-dev-tools

* https://github.com/Azure/azure-cli/blob/master/doc/extensions/authoring.md
