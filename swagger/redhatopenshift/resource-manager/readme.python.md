## Python

These settings apply only when `--python` is specified on the command line.

```yaml $(python)
python:
  azure-arm: true
  license-header: MICROSOFT_MIT_NO_VERSION
  payload-flattening-threshold: 2
  package-name: azure-mgmt-redhatopenshift
  clear-output-folder: true
  no-namespace-folders: true
```

### Python multi-api

Generate all API versions currently shipped for this package

```yaml $(python) && $(multiapi)
batch:
  - tag: package-2020-04-30
  - tag: package-2021-09-01-preview
  - tag: package-2022-04-01
```

### Tag: package-2020-04-30 and python

These settings apply only when `--tag=package-2020-04-30 --python` is specified on the command line.
Please also specify `--python-sdks-folder=<path to the root directory of your azure-sdk-for-python clone>`.

``` yaml $(tag) == 'package-2020-04-30' && $(python)
python:
  namespace: azure.mgmt.redhatopenshift.v2020_04_30
  output-folder: $(python-sdks-folder)/redhatopenshift/azure-mgmt-redhatopenshift/azure/mgmt/redhatopenshift/v2020_04_30
```

### Tag: package-2021-09-01-preview and python

These settings apply only when `--tag=package-2021-09-01-preview --python` is specified on the command line.
Please also specify `--python-sdks-folder=<path to the root directory of your azure-sdk-for-python clone>`.

``` yaml $(tag) == 'package-2021-09-01-preview' && $(python)
python:
  namespace: azure.mgmt.redhatopenshift.v2021_01_31_preview
  output-folder: $(python-sdks-folder)/redhatopenshift/azure-mgmt-redhatopenshift/azure/mgmt/redhatopenshift/v2021_01_31_preview
```

### Tag: package-2022-04-01 and python

These settings apply only when `--tag=package-2022-04-01 --python` is specified on the command line.
Please also specify `--python-sdks-folder=<path to the root directory of your azure-sdk-for-python clone>`.

``` yaml $(tag) == 'package-2022-04-01' && $(python)
python:
  namespace: azure.mgmt.redhatopenshift.v2022_04_01
  output-folder: $(python-sdks-folder)/redhatopenshift/azure-mgmt-redhatopenshift/azure/mgmt/redhatopenshift/v2022_04_01
```
