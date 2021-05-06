# Portal UI

## Developing

You will require Node.js and `npm`. These instructions were tested with the versions from the Fedora 34 repos.

1. Make your desired changes in `portal/src/` and commit them.

1. Run `make build-portal` from the main directory. This will install the dependencies and kick off the Webpack build, placing the results in `portal/dist/`.

1. Run `make generate`. This will regenerate the golang file containing the portal content to be served.

1. Commit the results of `build-portal` and `generate`.
