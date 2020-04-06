# RP Deployer tooling

Resource providers deployment automation is done using `deployer` tooling.

## Run
Deployer tooling can executed using command `aro deploy config mode location`,
where:

`config` - configuration file location. Example `./config.yaml.example`
`location` - location of the resource provider
`mode` - execution mode. Possible options are:
* `p` - predeploy
* `d` - deploy
* `u` - upgrade

Additional flag `f` configures `fullDeploy` mode.
`fullDeploy` mode will deploy all resources in the templates. If not
set it will deploy ONLY resource required for Upgrade. This can be run with lower
privilege ServicePrincipal.

## Examples

```
aro deploy ./config.yaml pf eastus - run predeploy in fullDeploy mode
aro deploy ./config.yaml duf eastus - run deploy and upgrade in fullDeploy mode
aro deploy ./config.yaml u eastus - run upgrade mode only
```
