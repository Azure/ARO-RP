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
az -v
