#!/bin/bash

set -o errexit

# Variables checked for timeout values, defaults used if unset
soft_timeout="${GO_GENERATE_SOFT_TIMEOUT:-2h}"
hard_timeout="${GO_GENERATE_HARD_TIMEOUT:-2h}"

if [[ -n $CI ]]; then
    set -x
fi


declare timeout_exitcode go_generate
# go generate doesn't support timeout itself, so --preserve-status wouldn't help us as it would simply be the status timeout sent to go.
go_generate="go generate ./..."
echo "Running ${go_generate} sending SIGTERM after ${soft_timeout} and SIGKILL after ${hard_timeout}"
echo "${go_generate}"
# shellcheck disable=SC2086
timeout --foreground --kill-after="${hard_timeout}" -v -s SIGTERM "${soft_timeout}" ${go_generate}
timeout_exitcode=$?

# Capture expected exit codes
case $timeout_exitcode in
124)
    echo "Command ${go_generate} timed out."
    ;;
125)
    echo "The command \"timeout\" failed trying to run ${go_generate}. Note that ${go_generate} did not fail."
    ;;
126)
    echo "Command found but cannot be invoked."
    ;;
127)
    echo "Command not found recieved."
    ;;
137)
    echo "timeout or command recieved kill signal 9) SIGKILL."
    ;;
-)
    echo "Command go generate ./... exited with exit_code code ${-}."
    ;;
0)
    echo "Command ${go_generate} completed successfully, exit code ${timeout_exitcode}."
    ;;
*)
    echo "Unexpected error code received: ${timeout_exitcode}."
    ;;
esac

exit "${timeout_exitcode}"
