#!/bin/bash -e
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
make az
mkdir -p ~/.azure
cat > ~/.azure/config <<EOF
[cloud]
name = AzureCloud

[extension]
dev_sources = $PWD/python
EOF
rm -f ~/.azure/commandIndex.json # https://github.com/Azure/azure-cli/issues/14997
az -v
