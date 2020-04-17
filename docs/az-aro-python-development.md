# How-to `az aro` development

ARO-RP distributed the CLI command `az aro` to control the cluster. 
Currently the `aro` subcommand is merged in the 
[azure-cli](https://github.com/azure/azure-cli) code base and 
is present as a command.

Currently, there are two codebases for `aro` subcommand:

* [upstream](https://github.com/Azure/ARO-RP/tree/master/python/az/aro) in main ARO-RP repo - which is an `az` extension
* [downstream][https://github.com/Azure/azure-cli/tree/dev/src/azure-cli/azure/cli/command_modules/aro] - which is an `az` command

Command is distributed alongside `az` and is automaticaly present in Azure cloud shell. Extension is installed as user action.

Currently code have to be in sync for upstream and downstream.

## Set up dev environment `az` for `aro` command

Python virtual environment is required:

```
python3 -m venv env
source <az dir>/env/bin/activate
```

in the `az` project directory works the best in case of command development.

To work with `az` helper tool `azdev` comes in handy

```
pip install azdev
```

`azdev` tool **runs only** in the virtual environment.

Then run

```
azdev setup -c
```

and follow the guide.

Once setup, the work on tool can be started.

## Code

`aro` subcommand is placed in

```
.../azure-cli/src/azure-cli/azure/cli/command_modules/aro
```

with tests in 

```
.../azure-cli/src/azure-cli/azure/cli/command_modules/aro/tests
```

There is wery useful guide for authoring commands in the
[azure-cli](https://github.com/Azure/azure-cli/tree/dev/doc/authoring_command_modules) repository.

## Tests

Tests can be run with

```
azdev test aro (--live) (--verbose) (--debug) 
```

there are two main types of tests:

* live tests, that gets recorded and replayed
* mocks

The guide for writting and operating the tests is in the
[azure-cli](https://github.com/Azure/azure-cli/blob/dev/doc/authoring_tests.md) repository.

### Live tests and recording

Tests can be recorded live, which means next run will be run against
the recorded values. For the recording python VCR library is used, and
recorded "tapes" are stored in the directory:

```
.../azure-cli/src/azure-cli/azure/cli/command_modules/aro/tests/latest/recordings
```

The test are recorded when the parameter `--live` is present, execution is forced to perform againts live Azure API. Every call is than recorded.

Recorded tests can be run as normal tests without the `--live`.

## Caveats

CI seems to not be isolated. Every run can fail or pass depending on the state of `dev` branch. Force push to rerun the tests.

Rebase has the potential to butcher the `venv`. Delete and recreate it. This solves the issues.

Rebases are a must as the dependencies are downloaded in wrong versions and even local environemnt tend to break against the API.

When designing tests for live recording, care need to be taken. UUIDs
are changing in the recording to mask the fact that it is a recording.