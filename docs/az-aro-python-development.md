# `az aro` Python Development

> [!IMPORTANT]
> **Where did the `python/` directory go?**
>
> The `az aro` development and preview extension has been moved into the [azure-cli-extensions](https://github.com/Azure/azure-cli-extensions) repo. The primary benefit for doing this is we get automated builds and published artifacts of the extension in a preview capacity for any customers that may want it.
>
> Please read more below for an explanation of how to develop the `az aro` development and preview extension and downstream command module.

There are two codebases for the `az aro` command:

- the [upstream] `az aro` development and preview extension that lives in azure-cli-extensions, and
- the [downstream] `az aro` command module in Azure CLI.

[upstream]: https://github.com/Azure/azure-cli-extensions/tree/main/src/aro
[downstream]: https://github.com/Azure/azure-cli/tree/dev/src/azure-cli/azure/cli/command_modules/aro

The downstream `az aro` command module is distributed with Azure CLI and is present in the Azure Cloud Shell. Development/maintenance of this module within Azure CLI is treated as a first-party Azure CLI module. Please see [Azure CLI documentation] for more information on command module authoring and maintenance.

[downstream documentation]: https://github.com/Azure/azure-cli/tree/dev/doc/authoring_command_modules

The upstream extension may be installed by an end user to override the command module (for example: to use preview features). ARO SRE uses the extension for development and testing of new API versions. Customers are advised to use the downstream command module included with Azure CLI. You can read more about [how extensions are authored](https://github.com/Azure/azure-cli/blob/dev/doc/extensions/authoring.md) and about some of the [differences between extensions and command modules](https://github.com/Azure/azure-cli/blob/dev/doc/extensions/faq.md).

ARO SRE aims for the upstream and downstream codebases to be as synchronized as possible.

> [!NOTE]
> On occasion, we may need to generate a new SDK client (i.e. - to add a new feature). See [API version documentation](api-versions.md) for more information on SDK client generation.

## Dev environment setup for the `az aro` **downstream command module**

```sh
git clone https://github.com/Azure/azure-cli.git
cd azure-cli

python3 -m venv env
source ./env/bin/activate
pip install --upgrade pip && pip install azdev
azdev setup -c .
```

The `az aro` command module is located in `src/azure-cli/azure/cli/command_modules/aro` with tests in the corresponding `tests` directory.

## Dev environment setup for the `az aro` **upstream development and preview extension**
<a href="#ext-dev"></a>

```sh
git clone https://github.com/Azure/azure-cli-extensions.git
cd azure-cli-extensions

python3 -m venv env
source ./env/bin/activate
pip install --upgrade pip && pip install azdev
azdev setup -r . -e aro

# the extension should show up as "installed" for the following command:
az extension list
```

The `az aro` development and preview extension is located in `src/aro/azext_aro` with tests in the corresponding `tests` folder.

## Upstream Azure CLI tests

Tests are run as follows:

```sh
azdev test azext_aro
```

> [!TIP]
> An issue was discovered on macOS when running tests due to additional security to restrict multithreading in macOS High Sierra and later versions of macOS.
>
> If you see the following error:
> ```
> +[__NSCFConstantString initialize] may have been in progress in another thread when fork() was called.
> ```
> Add this variable to your env or add it to your profile to make it permanent in `~/.bash_profile` or `~/.zshrc`: `export OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES`

There are two main types of tests:

- live tests that get recorded and replayed, and
- mocks

The guide for writing and operating the tests is in the [azure-cli](https://github.com/Azure/azure-cli/blob/dev/doc/authoring_tests.md) repository.

### Live tests and recording

Tests can be recorded live, which enables following runs to take place against the recorded values. For the recording, the Python VCR library is used; recorded "tapes" are stored in
`src/aro/azext_aro/tests/latest/{hcp,standard}/integration/recordings`.

When the `--live` flag is passed, the tests run against the Azure API and generate new recordings. Failed tests can be re-run using the `--lf` flag.

Before committing new recordings, sanitize them for UUIDs and email addresses.

## Contributing

Merge, publish, and approve changes to the `az aro` [upstream] development and preview extension **before** contributing the same changes the [downstream] command module. This allows us to keep the upstream extension and downstream command module as synchronized as possible.

When porting changes to the [downstream] command module, some imports will differ. When submitting pull requests, tests that include `azdev style`, `azdev linter`, and `azdev test azure-cli-aro` checks should pass to ensure that changes will not impact CI pipelines.

Additional instructions for merging changes downstream are contained in internal documentation.

## Caveats

> [!NOTE]
> Code in the upstream extension and downstream command module does not all share the same license. Microsoft code and Microsoft generated code (`aaz/` and `vendored_sdks/`) seem to generally be MIT, whereas most other code is Apache 2.0.

- Pulling `azure-cli` can break the python virtualenv. If this happens, delete and recreate it.
- Care needs to be taken when designing tests for live recording. The recording framework rewrites some fields, possibly including UUIDs.
