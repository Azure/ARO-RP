if ! command -v golangci-lint &> /dev/null
then
    echo "WARNING golangci-lint could not be found, golang lint skipped.To install it: https://golangci-lint.run/usage/install/#local-installation"
else
    golangci-lint run
fi
