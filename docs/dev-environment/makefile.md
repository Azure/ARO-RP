# Makefile Usage

### Adding New Makefile Targets

When adding new targets to the Makefile, ensure they follow the help-compatible format so they appear in the `make help` output. The help system uses a regex pattern to extract target names and descriptions.

**Format:** Add two hash marks (`##`) followed by a space and description after your target definition:

```makefile
target-name: dependencies ## Brief description of what this target does
	@commands to execute
```

**Example:**

```makefile
my-new-feature: install-tools ## Build and test my new feature
	go build ./pkg/myfeature
	go test ./pkg/myfeature/...
```

This target will then appear in the `make help` output as:

```sh
my-new-feature            Build and test my new feature
```

> [!IMPORTANT]
> The `##` delimiter must be present for the target to be recognized by the help system. Targets without this delimiter will not appear in the help output.

## Makefile Help
> [!IMPORTANT]
> Running `make init-contrib` enforces a necessary branch naming convention for your commits.
>
> The convention is: `<USERNAME>/<JIRA_NUMBER>`
> You can also append a description after the `<JIRA_NUMBER>` e.g: `<USERNAME>/<JIRA_NUMBER>/my-description-here`

> [!TIP]
> The project includes a `make help` target that provides a comprehensive list of all available Makefile targets along with their descriptions. This is particularly useful for discovering available commands without needing to read through the entire Makefile.

To view all available targets, run:
```sh
make help
```
