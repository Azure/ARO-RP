# `az aro` Python development

There are two codebases for the `az aro` command:

- the [downstream] `az aro` extension in this repo, and
- the [upstream] `az aro` module

[upstream]: https://github.com/Azure/azure-cli/tree/dev/src/azure-cli/azure/cli/command_modules/aro
[downstream]: https://github.com/Azure/ARO-RP/tree/master/python/az/aro

The upstream `az aro` command module is distributed with `az` and is present in
the Azure cloud shell. Development/maintenance of this module within the Azure
CLI is treated as a first-party Azure CLI module. Please see [upstream
documentation] for more info on command module authoring and maintenance.

[upstream documentation]: https://github.com/Azure/azure-cli/tree/dev/doc/authoring_command_modules

The downstream extension may be installed by an end user to override the command module
(for example: to use preview features). We use the extension directly from this
codebase for development and testing of new API versions. Customers are
advised to use the upstream CLI. You can read more about how extensions are
authored [here](https://github.com/Azure/azure-cli/blob/dev/doc/extensions/authoring.md),
and some of the differences between extensions and command modules
[here](https://github.com/Azure/azure-cli/blob/dev/doc/extensions/faq.md).

We aim for the upstream and downstream codebase to be as synchronized as
possible.

## Dev environment setup for the `az aro` **command**

```sh
git clone https://github.com/Azure/azure-cli.git
cd azure-cli

python3 -m venv env
. ./env/bin/activate
pip install azdev
azdev setup -c
```

The `az aro` command is located in `src/azure-cli/azure/cli/command_modules/aro`
with tests in the corresponding `tests` folder.


## Dev environment setup for the `az aro` **extension**

```sh
git clone https://github.com/Azure/ARO-RP.git
cd ARO-RP

make pyenv
source ./pyenv/bin/activate
```

The `az aro` extension is located in `python/az/aro/azext_aro` with tests in
the corresponding `tests` folder.

There is a useful guide for authoring commands in the
[azure-cli](https://github.com/Azure/azure-cli/tree/dev/doc/authoring_command_modules)
repository.

## `az aro` Extension Code Structure/Organization

The majority of hand-written code lives in `python/az/aro/azext_aro`:

- `python/az/aro/azext_aro/__init__.py` - ARO extension entrypoint
- `python/az/aro/azext_aro/commands.py` - ARO extension command structure
  definitions
- `python/az/aro/azext_aro/custom.py` - Logic and helper methods for
  subcommands
- `python/az/aro/azext_aro/_help.py` - Help output definitions

### Generated code

Helpful to reference, but these don't need edits:

- `python/az/aro/azext_aro/aaz` - Generated code vendored from AZ tooling that
  we occasionally change new classes or functions are needed.
- `python/az/aro/build`
- `python/az/client`

## Tests

Tests are run as follows:

```bash
azdev test aro (--live) (--lf) (--verbose) (--debug)
```

> [!TIP]
> An issue was discovered on macOS when running tests due to additional
> security to restrict multithreading in macOS High Sierra and later versions
> of macOS.
>
> If you see the following error:
> ```
> +[__NSCFConstantString initialize] may have been in progress in another
> thread when fork() was called.
> ```
> Add this variable to your env or add it to your profile to make it permanent
> in `~/.bash_profile` or `~/.zshrc`: `export
> OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES`


There are two main types of tests:

- live tests that get recorded and replayed, and
- mocks

The guide for writing and operating the tests is in the
[azure-cli](https://github.com/Azure/azure-cli/blob/dev/doc/authoring_tests.md)
repository.


### Live tests and recording

Tests can be recorded live, which enables the next run to take place against
the recorded values. For the recording, the Python VCR library is used;
recorded "tapes" are stored in
`src/azure-cli/azure/cli/command_modules/aro/tests/latest/recordings`.

When the `--live` flag is passed, the tests run against the Azure API and are
recorded. Failed tests can be re-run using the `--lf` flag.

Recorded tests are run when the `--live` flag is not passed.

## Contributing

Changes made to the `az aro` command will be made to `Azure/ARO-RP` **before**
being contributed [upstream]. This will allow synchronization between the two
repositories to be consistent. Once changes are made and approved to
[downstream], pull requests to upstream can then be opened.

When contributing to the `az aro` command upstream, imports will be different
for testing. When submitting pull requests, tests that include `azdev style`,
`azdev linter`, and `azdev test aro` should pass to ensure that changes will not
impact CI pipelines.

Additional instructions for merging changes upstream are contained in internal
documentation.

## Updating Azure CLI version

When upstream azure-cli releases a new version that we need to adopt, follow
these steps to regenerate `requirements.txt`:

```sh
# 1. Remove existing pyenv
rm -rf pyenv

# 2. Create fresh virtual environment
python3 -m venv pyenv

# 3. Install new azure-cli version + azdev
. pyenv/bin/activate
pip install -U pip
pip install azure-cli==<NEW_VERSION> azdev

# 4. Setup azdev with the extension repo
azdev setup -r .

# 5. Freeze dependencies to regenerate requirements.txt
pip freeze > requirements.txt
```

Also update `Dockerfile.ci-azext-aro` to match the new CLI version (both the
pip install line and the base image).

This lets pip resolve all SDK dependencies automatically rather than manually
editing individual packages.

## Caveats

- `azure-cli` CI is not entirely isolated.  Test runs may pass or fail
  depending on the state of `dev` branch.  Force push to rerun the tests.
- Pulling `azure-cli` can break the `venv`.  If this happens, delete and
  recreate it.
- Care needs to be taken when designing tests for live recording. The recording
  framework rewrites some fields, possibly including UUIDs.
- When developing the `az aro` extension in this repository, you may wish to
  use the [edge](https://github.com/Azure/azure-cli#edge-builds) CLI version to
  be as close as possible to azure-cli master.
