# Upstream differences

This file catalogues the differences of install approach between ARO and
upstream OCP.

## Installer carry patches

See https://github.com/openshift/installer/compare/release-4.9...jewzaam:release-4.9-azure.

## Installation differences

* ARO persists the install graph in the cluster storage account in a new "aro"
  container / "graph" blob.

* No managed identity (for now).

* No IPv6 support (for now).

* Upstream installer closely binds the installConfig (cluster) name, cluster
  domain name, infra ID and Azure resource name prefix.  ARO separates these out
  a little.  The installConfig (cluster) name and the domain name remain bound;
  the infra ID and Azure resource name prefix are taken from the ARO resource
  name.

* API server public IP domain name label is not set.

* ARO uses first party RHCOS OS images published by Microsoft.

* ARO never creates xxxxx-bootstrap-pip-* for bootstrap VM, or the corresponding
  NSG rule.

* ARO does not create a outbound-provider Service on port 27627.

* ARO deploys a private link service in order for the RP to be able to
  communicate with the cluster.

* ARO runs a dnsmasq service on the nodes through the use of a machineconfig to resolve api-int and *.apps domains on the node locally allowing for custom DNS configured on the VNET.

# Introducing new OCP release into ARO RP

## Update installer fork

To bring new OCP release branch into ARO installer fork:

1. Check git diff between the target release branch release-X.Y and previous one release-X.Y-1
   to see if any resources changed and/or architecture changed.
   These changes might require more modifications on ARO-RP side later on.
1. Create a new release-X.Y-azure branch in the ARO installer fork from upstream release-X.Y branch.
1. Cherry-pick all commits from the previous release-X.Y-1-azure branch into the new one & fix conflicts.
    * While cherry-picking `commit data/assets_vfsdata.go` commit, run `cd ./hack/assets/ && go run ./assets.go`
    to generate assets and then add them to this commit.
1. Run `./hack/build.sh` and `./hack/go-test.sh` as part of every commit (`git rebase` with `-x` can help with this).
    * Fix build and test failures.

**Note:** If any changes are required during the process, make sure to amend the relevant patch or create a new one.
Each commit should be atomic/complete - you should be able to cherry-pick it into the upstream installer and bring
the fix or feature it carries in full, without a need to cherry-pick additional commits.
This makes it easier to understand the nature of the patch as well as contribute our carry patches
back to the upstream installer.

# Update ARO-RP

Once installer fork is ready:

1. Update `go mod edit -replace` calls in `hack/update-go-module-dependencies.sh` to use a new release-X.Y branch.
    * Make sure to read comments in the script.
1. `make vendor`.
    * You most likely will have to make changes to the RP codebase at this point to adjust it to new versions of dependencies.
    * Also you likely will have to repeat this step several time until you resolve all conflicting dependencies.
      Follow `go mod` failures, which will tell you what module requires what other module.
      You will probably need to look at the `go.mod` files of these modules and see whether they set own replace directives,
      as the script is likely to fail with something like this:

      ```
      go: github.com/openshift/installer@v0.16.1 requires
          github.com/openshift/cluster-api-provider-kubevirt@v0.0.0-20201214114543-e5aed9c73f1f requires
          kubevirt.io/client-go@v0.0.0-00010101000000-000000000000: invalid version: unknown revision 000000000000
      ```

      In the example above you need to:
        * Checkout `github.com/openshift/cluster-api-provider-kubevirt` at commit `e5aed9c73f1f`.
        * In go.mod find a replace directive for `kubevirt.io/client-go`.
        * Add/update relevant replace directive in ARO-RP `go.mod`.
1. `make generate`.
1. Update `pkg/util/version/const.go` to point to the new release.
    * You should be able to find latest published release and image hash [on quay.io](https://quay.io/repository/openshift-release-dev/ocp-release?tab=tags).
1. Publish RHCOS image. See [this document](./publish-rhcos-image.md).
1. After this point, you should be able to create a dev cluster using the RP and it should use the new release.
1. `make discoverycache`.
    * This command requires a running cluster with the new version.
