# `az aro` Python development

There are currently two codebases for the `az aro` command:

* The [upstream](https://github.com/Azure/ARO-RP/tree/master/python/az/aro) `az
  aro` extension in this repo

* The
  [downstream][https://github.com/Azure/azure-cli/tree/dev/src/azure-cli/azure/cli/command_modules/aro]
  `az aro` module.

The downstream `az aro` module is distributed with `az` and is automatically
present in the Azure cloud shell.  The upstream extension can be installed by an
end user to override the module (e.g. to fix an issue rapidly).  We use the
extension for development and testing of new API versions.

We aim for the upstream and downstream codebase to be as closely in sync as
possible.


## Dev environment setup for the `az aro` command

```bash
git clone https://github.com/Azure/azure-cli.git
cd azure-cli

python3 -m venv env
. ./env/bin/activate
pip install azdev
azdev setup -c
```

The `az aro` command is located in `src/azure-cli/azure/cli/command_modules/aro`
with tests in corresponding `tests` folder.


## Dev environment setup for the `az aro` extension

```bash
git clone https://github.com/Azure/ARO-RP.git
cd ARO-RP

make pyenv
```

The `az aro` extension is located in `python/az/aro/azext_aro` with tests in
corresponding `tests` folder.

There is a very useful guide for authoring commands in the
[azure-cli](https://github.com/Azure/azure-cli/tree/dev/doc/authoring_command_modules)
repository.


## Tests

Tests can be run as follows:

```bash
azdev test aro (--live) (--lf) (--verbose) (--debug)
```

There are two main types of tests:

* live tests that get recorded and replayed
* mocks

The guide for writing and operating the tests is in the
[azure-cli](https://github.com/Azure/azure-cli/blob/dev/doc/authoring_tests.md)
repository.


### Live tests and recording

Tests can be recorded live, which enables the next run to take place against the
recorded values.  For the recording, the Python VCR library is used; recorded
"tapes" are stored in
`src/azure-cli/azure/cli/command_modules/aro/tests/latest/recordings`.

When the `--live` flag is passed, the tests run against the Azure API and are
recorded.  Failed tests can be re-run using the `--lf` flag.

Recorded tests are run when the `--live` flag is not passed.


## Caveats

* `azure-cli` CI is not entirely isolated.  Test runs may pass or fail depending
  on the state of `dev` branch.  Force push to rerun the tests.

* Pulling `azure-cli` can break the `venv`.  If this happens, delete and
  recreate it.

* Care needs to be taken when designing tests for live recording.  The recording
  framework rewrites some fields, possibly including UUIDs.

* When developing the `az aro` extension in this repository, you may wish to use
  the [edge](https://github.com/Azure/azure-cli#edge-builds) CLI version to be
  as close as possible to azure-cli master.
