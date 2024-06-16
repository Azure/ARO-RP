# Updating dependencies

To update golang dependencies one has to choose between two options:

1) update the whole dependency tree
2) update or add what is currently required


## Update the whole dependency tree

Done when the need to catch upstream dependencies arises, it can be done by
calling

```bash
make vendor
```

in root folder, which calls `hack/update-go-module-dependencies.sh`.

The reason for calling script instead of directly calling:

```bash
go get -u ./...
make go-verify
```

is that packages modified in this script do not fully support modules and
semantic versioning via tags. Therefore the proper version is parsed from the version
branch and fixed using replace directive. Otherwise it will upgrade every time
the command is started.

When upgrading to a newer version of OpenShift, this script have to be updated to
reflect the proper release.


## Update only required dependencies

When adding new dependencies or updating old ones, based on the requirement of
the PR, one can simply call

```bash
go get <module>@<release> OR
go get -u <module>@<release>

make go-verify
```

---

**NOTE** vendoring is required as ARO mirrors all dependencies locally for the CI reliability
and reproducibility.

**NOTE** that running `make go-verify` adds missing (and remove unused) modules,
make a vendored copy to the *vendor* direcory and then verifies that the dependencies have expected content.

**NOTE** when updating a package modified in `hack/update-go-module-dependencies.sh`
changes have to be made there also. Otherwise next run of `make vendor` will
overwrite new changes.

More information on working with modules: https://blog.golang.org/using-go-modules
