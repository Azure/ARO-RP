# Pipelines

CI is running on the Azure Dev Ops Pipelines, which is configured in the
`.pipelines` directory.

Documentation for the pipelines is located in the [MSDN](https://docs.microsoft.com/en-us/azure/devops/pipelines/?view=azure-devops).


## Pipelines triggers

**e2e** are configured to be explicitly run when new commit is done on `master` branch, excluding changes for files in `docs/*`.
Default pipelines configuration is implicitly doing this, but for all folders.

For PRs the **e2e** is triggered also when it should be merged to the `master`, but excludes changes in `docs/*`.

This saves testing infrastructure cycles for simple documentation updates.