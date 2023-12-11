# Updating dependencies

  - Go Documentation for [Managing dependencies](https://go.dev/doc/modules/managing-dependencies)
  - [Go Module Awareness](https://go.dev/blog/go116-module-changes)

To update golang dependencies one has to choose between two options:

1) update the whole dependency tree
2) update or add what is currently required


## Update the whole dependency tree

Done when the need to catch upstream dependencies arises, it can be done by
calling:

```bash
# Ensure go module-aware mode is set to auto
export GO111MODULE=auto
go get -u ./...
go mod tidy -compat=1.18
go mod vendor
```

## Update only required dependencies

When adding new dependencies or updating old ones, based on the requirement of
the PR, one can simply call:

```bash
export GO111MODULE=auto
go get <module>@<release> OR
go get -u <module>@<release>

go mod tidy -compat=1.18
go mod vendor
```

---

**NOTE** vendoring is required as ARO mirrors all dependencies locally for the CI reliability
and reproducibility.

**NOTE** that when running `go mod vendor` only modules that are used in the
source code will be vendored in.

**NOTE** when updating a package modified in `hack/update-go-module-dependencies.sh`
changes have to be made there also. Otherwise next run of `make vendor` will
overwrite new changes.

More information on working with modules: https://blog.golang.org/using-go-modules
