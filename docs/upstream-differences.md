# Upstream differences

This file catalogues the differences of install approach between ARO and
upstream OCP.

## Installer carry patches

1. Clone ARO and upstream repos:
    ```sh
    # clone our forked installer
    git clone https://github.com/openshift/installer-aro.git
    cd installer-aro

    # add the upstream as a remote source
    git remote add upstream https://github.com/openshift/installer.git
    git fetch upstream -a
    ```
1. See carry patches from previous release:
    ```sh
    # list patches
    git log upstream/release-X.Y-1..origin/release-X.Y-1-azure

    # see diff of patches
    git show upstream/release-X.Y-1..origin/release-X.Y-1-azure
    ```

## Installation differences

* ARO does not use Terraform to create clusters, and instead uses ARM templates directly

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

To support a new version of OpenShift on ARO, you will need to reconcile [upstream changes](https://github.com/openshift/installer) with our [forked installer](https://github.com/openshift/installer-aro). This will not be a merge, but a cherry-pick of patches we've implemented.

## Update installer fork

To bring new OCP release branch into ARO installer fork:

1. If not done already, fetch our fork and upstream repos:
    ```sh
    # clone our forked installer
    # Alternatively, fork openshift/installer-aro and clone your fork
    git clone https://github.com/openshift/installer-aro.git
    cd installer-aro
    
    # add the upstream as a remote source
    git remote add upstream https://github.com/openshift/installer.git
    git fetch upstream -a
    ```
1. Assess and document differences in X.Y and X.Y-1 in upstream
    ```sh
    # diff the upstream X.Y with X.Y-1 and search for architecture changes
    git diff upstream/release-X.Y-1 upstream/release-X.Y
    
    # pay particular attention to Terraform files, which may need to be moved into ARO's ARM templates
    git diff upstream/release-X.Y-1 upstream/release-X.Y */azure/*.tf
    ```
2. Create a new X.Y release branch in our forked installer
    ```sh
    # create a new release branch in the fork based on the upstream
    git checkout upstream/release-X.Y
    git checkout -b release-X.Y-azure
    ```
3. If there is a golang version bump in this release, modify `./hack/build.sh` and `./hack/go-test.sh` with the new version, then verify these scripts still work and commit them
4. Determine the patches you need to cherry-pick, based on the last (Y-1) release
    ```sh
    # find commit shas to cherry-pick from last time
    git checkout release-X.Y-1-azure
    git log
    ```
5. For every commit you need to cherry-pick (in-order), do:
    ```sh
    # WARNING: when you reach the commit for `commit data/assets_vfsdata.go`, look ahead
    git cherry-pick abc123 # may require manually fixing a merge
    ./hack/build.sh   # fix any failures
    ./hack/go-test.sh # fix any failures
    # if you had to manually merge, you can now `git cherry-pick --continue`
    ```
    - When cherry-picking the specific patch `commit data/assets_vfsdata.go`, instead run:
        ```sh
        git cherry-pick abc123 # may require manually fixing a merge
        ./hack/build.sh   # fix any failures
        ./hack/go-test.sh # fix any failures
        # if you had to manually merge, you can now `git cherry-pick --continue`
        pushd ./hack/assets && go run ./assets.go && popd
        ./hack/build.sh   # fix any failures
        ./hack/go-test.sh # fix any failures
        git add data/assets_vfsdata.go
        git commit --amend
        ```

**Note:** If any changes are required during the process, make sure to amend the relevant patch or create a new one.
Each commit should be atomic/complete - you should be able to cherry-pick it into the upstream installer and bring
the fix or feature it carries in full, without a need to cherry-pick additional commits.
This makes it easier to understand the nature of the patch as well as contribute our carry patches
back to the upstream installer.

## Update Installer Fork Again

In the far or near future after you have [initially patched the installer](#update-installer-fork), you may need to pull in additional upstream changes that have happened since you patched the installer (e.g. upstream added a bugfix since your cherry-picking). The easiest way to pull in these changes safely is:

```sh
git fetch upstream -a
git checkout release-X.Y-azure
git pull upstream/release-X.Y --rebase=interactive
```

When you get to the editor mode to set up your rebase, you should do a few things:

1. after each `pick` line, add the verification commands with:
    ```text
    exec ./hack/build.sh
    exec ./hack/go-test.sh
    ```
1. change the line for commit `data/assets_vfsdata.go` from `pick` to `edit` so you can regenerate assets as outlined [above](#update-installer-fork), and verify it manually.

By the end of this editing process, you should have a rebase file that looks something like:

```text
pick abc123 patch A
exec ./hack/build.sh
exec ./hack/go-test.sh
pick def234 patch B
exec ./hack/build.sh
exec ./hack/go-test.sh
edit ghi345 patch data/assets_vfsdata.go
exec ./hack/build.sh
exec ./hack/go-test.sh
pick jkl456 patch C
exec ./hack/build.sh
exec ./hack/go-test.sh
{...}
```

When you are finished, you can write/close the file editor to perform the rebase on the upstream differences while verifying that every patch still works along the way.

Get a repo admin to create `release-X.Y-azure` in [openshift/installer-aro](https://github.com/openshift/installer-aro). You can now push your local branch `release-X.Y-azure` to your own fork of [openshift/installer-aro](https://github.com/openshift/installer-aro) and send a PR.

# Update ARO Installer Wrapper and ARO-RP

Once installer fork is ready, perform the following changes in the [ARO Installer Wrapper](https://github.com/openshift/installer-aro-wrapper):

1. Update `go mod edit -replace` calls in `hack/update-go-module-dependencies.sh` to use a new release-X.Y branch.
    * Make sure to read comments in the script.
1. `make vendor`.
    * You most likely will have to make changes to the codebase at this point to adjust it to new versions of dependencies.
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
        * Add/update relevant replace directive in `go.mod`.
1. `make generate`.
1. A local image can be built for testing purposes and pushed to a dev ACR repo. The process for [publishing the final image uses ADO](https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/452838/ARO-Installer-Image-Deployment-Process)

In the ARO-RP codebase:

1. Update `pkg/util/version/const.go` to point to the new release.
    * You should be able to find latest published release and image hash [on quay.io](https://quay.io/repository/openshift-release-dev/ocp-release?tab=tags).
1. (Optional) `make discoverycache`.
    * This command requires a running cluster with the new version.
1. The list of the hard-coded namespaces in `pkg/util/namespace/namespace.go` needs to be updated regularly as every
   minor version of upstream OCP introduces a new namespace or two.


Publish RHCOS image to the Azure Cloud Partner Portal:
1. Publish RHCOS image. See [this document](https://github.com/openshift/installer-aro-wrapper/blob/main/docs/publish-rhcos-image.md).
1. After this point, you should be able to create a dev cluster using the RP and it should use the new release.
