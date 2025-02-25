#!/usr/bin/env bash

function help () {
    echo "List non-production git tags"
    echo
    echo "USAGE:"
    echo "    list-prune-tags.sh [OPTIONS] [EXCLUDE_TAG ...]"
    echo
    echo "OPTIONS:"
    echo "    -h           Display this help output and exit."
    echo "    -n           Do not print a newline after each item."
    echo
    echo "PARAMETERS:"
    echo "    EXCLUDE_TAG  (optional) Exclude this tag from the list of printed"
    echo "                 tags. Multiple tags can be excluded."
}

function main () {
    local -ar skip_tags=( $@ )
    local -ar nonproduction_tags=( $(git tag -l | grep -v -E '^v[[:digit:]]{8}.[[:digit:]]{1,2}$') )
    local -a output

    local tag
    for tag in ${nonproduction_tags[*]}; do
        if [ -n "${skip_tags[*]}" ] && [[ " ${skip_tags[*]} " =~ " $tag " ]]; then
            continue
        fi

        output+=( $tag )
    done

    if [ "${NEWLINE:-true}" == "true" ]; then
        for tag in ${output[*]}; do
            printf $tag"\n"
        done
    else
        echo ${output[*]}
    fi
}

while getopts ":hn" opt; do
    case "$opt" in
        h)
            help && exit
            ;;
        n)
            NEWLINE=false
            ;;
        ?)
            help && exit
            ;;
    esac
done
shift $((OPTIND - 1))

main "$@"
