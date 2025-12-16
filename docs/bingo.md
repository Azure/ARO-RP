# Managing Development Tools with Bingo

We use [bingo](https://github.com/bwplotka/bingo) to manage our build and testing tools independently of system installations. This ensures that all developers and the build pipeline use consistent versions of build tools.

**Notable tools managed by Bingo include:**
- [client-gen](https://github.com/kubernetes/code-generator/tree/master/cmd/client-gen)
- [controller-gen](https://github.com/kubernetes-sigs/controller-tools)
- [golangci-lint](https://github.com/golangci/golangci-lint)
- [gotestsum](https://github.com/gotestyourself/gotestsum)
- [mockgen](https://github.com/uber-go/mock)

You can view the full list of managed tools using the `bingo` command-line tool or by inspecting the `*.mod` files in the `.bingo/` directory.

## Common Tasks

### Installing Bingo CLI

Bingo itself is managed by Bingo. To install the CLI (and all other tools needed), run:

```sh
make install-tools

# Check the installed version
bingo version
v0.9
```

### Adding a New Tool

To add a new tool to be managed by Bingo, run:

```sh
bingo get -l [package]@[version|latest]
```

Replace `[package]@[version]` with the Go tool and version you wish to install, similar to `go install ...`. For example:

```sh
bingo get -l github.com/fatih/faillint@latest
bingo get -l github.com/golangci/golangci-lint/v2@v2.2.1
```

After adding a tool, commit and push the changes in the `.bingo` directory to a new branch and open a pull request for review.

> **Note:**
> Using the `-l` flag with `bingo get` creates a symlink to the installed tool in `$GOBIN`.

### Updating an Existing Tool

To update a tool, use the same process as adding a new one. For example:

```sh
bingo get -l github.com/golangci/golangci-lint/v2@latest
```

After updating, commit and push the changes and open a pull request.


