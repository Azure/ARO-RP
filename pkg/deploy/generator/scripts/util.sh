#!/bin/bash
# File to be sourced by *VMSS.sh scripts
# This is only present for the ability to manaully run the VMSS setup scripts seperate from the deploy process.
# e. g. scp copying the script to a test VM
# During normal deployment operations, the other util-*.sh files are prefixed to the VMSS scripts

if [ "${DEBUG:-false}" == true ]; then
    set -x
fi

util_common="util-common.sh"
if [ -f "$util_common" ]; then
    # shellcheck source=util-common.sh
    source "$util_common"
fi

util_system="util-system.sh"
if [ -f "$util_system" ]; then
    # shellcheck source=util-system.sh
    source "$util_system"
fi

util_services="util-services.sh"
if [ -f "$util_services" ]; then
    # shellcheck source=util-services.sh
    source "$util_services"
fi

util_pkgs="util-packages.sh"
if [ -f "$util_pkgs" ]; then
    # shellcheck source=util-packages.sh
    source "$util_pkgs"
fi
