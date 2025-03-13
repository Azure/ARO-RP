#!/usr/bin/env bash

# in days
declare -r MIN_TAG_AGE=60

function help () {
    echo
    echo "List non-production git tags older than ${MIN_TAG_AGE} days"
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
    local -r datef="%Y%m%d"

    # FIXME: this will break once we enter the year 10,000.
    local -ar persist_filters=(
        "^v[0-9]{8}\.[0-9]{1,2}$"
        "^azext-aro-v[0-9]{8}.*-[0-9]{1,2}$"
    )

    local -ar excluded_tags=( $@ )
    local -ar all_tags=( $(git tag -l) )
    local -a prune

    local tag
    for tag in ${all_tags[*]}; do
        # excluded tags get skipped
        if [ -n "${excluded_tags[*]}" ] && [[ " ${excluded_tags[*]} " =~ " $tag " ]]; then
            continue
        fi

        local filter match=false
        for filter in ${persist_filters[*]}; do
            [[ $tag =~ $filter ]] && match=true
        done

        [ "$match" == "true" ] && continue

        # annotated tag date
        local tag_date=$(git for-each-ref --format="%(taggerdate:format:$datef)" refs/tags/$tag)

        # fallback: date from the commit (if this is a lightweight tag)
        if [ -z "$tag_date" ]; then
            tag_date=$(git for-each-ref --format="%(creatordate:format:$datef)" refs/tags/$tag)
        fi

        # is the tag older than MIN_TAG_AGE?
        if [ ${tag_date} -lt $(date --date="${MIN_TAG_AGE} days ago" +$datef) ]; then
            prune+=( $tag )
        fi

        # no-op
    done

    if [ "${NEWLINE:-true}" == "true" ]; then
        for tag in ${prune[*]}; do
            printf $tag"\n"
        done
    else
        echo ${prune[*]}
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
