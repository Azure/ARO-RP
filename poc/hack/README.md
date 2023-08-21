# Scripts for ARO-RP on AKS POC

## build-deploy.sh
- Used to build and deploy ARO-RP from local environment
- Run this via the Makefile. 
- In the root ARO-RP directory, run `make alias=*your_alias* poc-build-deploy`
- This will upload your local RP image to the ACR as `dev/*your_alias*:latest`
- This will create a namespaces `*your_alias*-dev` where the local RP build will be deployed