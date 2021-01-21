#!/bin/bash

# - Switches between public and US Gov Azure clouds in a dev environment.
# - Overwrites the same symlink, 'secrets', pointing to whichever directory
#   holds the secrets for the current cloud environment.
# - Gets secrets, if they don't exist, when switching to a different or the same cloud.
# - Upon first run, backs up existing secrets directory as 'secrets-old'.

public_cloud_secrets_dir="secrets-public"
usgov_cloud_secrets_dir="secrets-usgov"

cloud=$(az cloud show --query 'name')
cloud="${cloud%\"}"
cloud="${cloud#\"}"

ensure_cloud_secret_dirs () {
  if [ ! -d $1 ] # if the cloud-specific secret dir doesn't exist
  then
    echo "Directory ./$1 does not exist; creating..."
    mkdir $1
  fi
}

ensure_cloud_secret_dirs $public_cloud_secrets_dir
ensure_cloud_secret_dirs $usgov_cloud_secrets_dir

is_empty () {
  if [ ! "$(ls -A secrets)" ] # if the secrets dir is empty...
  then
    echo "Secrets for $1 not populated. Running 'make secrets'..."
    SECRET_SA_ACCOUNT_NAME=$2 make secrets
  fi
}

switch_cloud () {
  echo "Switching to $1..."
  az cloud set --name $1
  az login
  if [ ! -L $2 ] # if ./secrets is a dir, not a symlink, rename it
  then
  	mv secrets secrets-old
  fi
  ln -sfn $2 secrets # if ./secrets symlink exists, overwrite it
  echo "Directory ./secrets now linked to ./$2"
  is_empty $1 $3
  echo ""
}

echo ""
echo "Current cloud is $cloud".
echo "Select an option:"
echo " 0. No change (or just press Enter)"
echo " 1. Switch to AzureCloud (AzurePublicCloud), get secrets if needed"
echo " 2. Switch to AzureUSGovernment, get secrets if needed"
echo -n "Your choice: "
read choice

case $choice in
  0|"")
    echo "No changes made."
    echo ""
    ;;
  1) # AzureCloud (AzurePublicCloud)
    switch_cloud AzureCloud $public_cloud_secrets_dir rharosecrets
    ;;
  2) # AzureUSGovernment
    switch_cloud AzureUSGovernment $usgov_cloud_secrets_dir rharogovsecrets
    ;;
  *)
    echo "Invalid option. No changes made."
    echo ""
    exit 1
esac
