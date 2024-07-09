#!/bin/bash
set -ex

extract_image_tag() {       
    # Extract the line containing the return statement
    local return_line=$(grep -A 1 "func $1" "$2" | grep 'return')
    echo $return_line | sed 's/.*"\(.*\)@sha256.*/\1/'
}
