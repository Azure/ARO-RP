#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

readonly repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly shellspec_image="${BASH_TEST_IMAGE:-shellspec/shellspec-debian:0.28.1}"
readonly report_dir="${BASH_TEST_REPORT_DIR:-/tmp/shellspec-report}"

container_engine_usable() {
    "$1" info >/dev/null 2>&1
}

detect_container_engine() {
    if command -v docker >/dev/null 2>&1 && container_engine_usable docker; then
        echo docker
        return 0
    fi

    if command -v podman >/dev/null 2>&1 && container_engine_usable podman; then
        echo podman
        return 0
    fi

    echo "error: a working docker or podman daemon is required to run bash tests" >&2
    return 1
}

detect_jobs() {
    if [[ -n "${BASH_TEST_JOBS:-}" ]]; then
        echo "${BASH_TEST_JOBS}"
        return 0
    fi

    if command -v nproc >/dev/null 2>&1; then
        nproc
        return 0
    fi

    if command -v getconf >/dev/null 2>&1; then
        getconf _NPROCESSORS_ONLN
        return 0
    fi

    if command -v sysctl >/dev/null 2>&1; then
        sysctl -n hw.ncpu
        return 0
    fi

    echo 4
}

detect_platform() {
    if [[ -n "${BASH_TEST_PLATFORM:-}" ]]; then
        echo "${BASH_TEST_PLATFORM}"
        return 0
    fi

    if [[ "$(uname -s)" == "Darwin" ]]; then
        echo "linux/amd64"
    fi
}

main() {
    local engine jobs mount_arg
    engine="$(detect_container_engine)"
    jobs="$(detect_jobs)"

    if [[ "${engine}" == "podman" ]]; then
        mount_arg="${repo_root}:/work:Z"
    else
        mount_arg="${repo_root}:/work"
    fi

    local -a platform_args=()
    local platform
    platform="$(detect_platform)"
    if [[ -n "${platform}" ]]; then
        platform_args=(--platform "${platform}")
    fi

    echo "Running ShellSpec with ${engine} using image ${shellspec_image}"

    "${engine}" run --rm \
        "${platform_args[@]}" \
        -v "${mount_arg}" \
        -w /work \
        "${shellspec_image}" \
        --jobs "${jobs}" \
        --format documentation \
        --output junit \
        --reportdir "${report_dir}" \
        "$@"
}

main "$@"
