#!/bin/bash

set -e

# shift off the first argument that we will use
COMMAND=$1
shift

# Load bingo's variables to get the build/run helpers
DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source $DIR/../.bingo/variables.env

${COMMAND} "$@"
