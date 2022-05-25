#!/bin/bash -e

# lint-go.sh is used to run Go linting. This shell script is invoked by the Makefile.
# It is not used in our pipelines as we have a github action that does the Go linting, so this script
# is intended to be used locally when developing.

# What does this script?
# It will try to execute the golangci-lint binary to check if there are any linter errors.
# In case the binary that is trying to execute does not exist,
# the script will print a message with instructions to download the binary and will exit with a non-zero exit code.

if ! command -v golangci-lint &> /dev/null
then
    echo "ERROR: golangci-lint could not be found, Go linting aborted. To install it visit https://golangci-lint.run/usage/install/#local-installation"
    exit 1
else
    golangci-lint run
fi