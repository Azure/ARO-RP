#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly repo_root
readonly shellspec_image="${BASH_TEST_IMAGE:-docker.io/shellspec/shellspec-debian:0.28.1}"
readonly default_report_base="${TMPDIR:-/tmp}/aro-bash-test-report"

# Reject report paths that could make host-side cleanup delete the wrong tree.
validate_report_dir() {
    local report_dir="$1"
    local tmpdir="${TMPDIR:-/tmp}"
    local path_segment

    tmpdir="${tmpdir%/}"

    if [[ -z "${report_dir}" || "${report_dir}" != /* || -z "${report_dir//\//}" ]]; then
        echo "error: refusing unsafe report directory '${report_dir}'" >&2
        return 1
    fi

    # Only allow report directories under the temp dir with the expected prefix.
    case "${report_dir}" in
        "${tmpdir}"/aro-bash-test-report*)
            ;;
        *)
            echo "error: refusing unsafe report directory '${report_dir}'" >&2
            return 1
            ;;
    esac

    IFS='/' read -r -a path_segments <<< "${report_dir#/}"
    for path_segment in "${path_segments[@]}"; do
        if [[ "${path_segment}" == "." || "${path_segment}" == ".." ]]; then
            echo "error: refusing unsafe report directory '${report_dir}'" >&2
            return 1
        fi
    done
}

# Recreate the host report directory that ShellSpec writes into from the container.
prepare_report_dir() {
    local report_dir
    report_dir="${BASH_TEST_REPORT_DIR:-${default_report_base}}"
    validate_report_dir "${report_dir}" || return 1
    rm -rf -- "${report_dir}"
    mkdir -p -- "${report_dir}"
    printf '%s\n' "${report_dir}"
}

# Treat an engine as usable only when its info command succeeds.
container_engine_usable() {
    "$1" info > /dev/null 2>&1
}

# Detect podman-docker wrappers so we can apply Podman-specific run flags.
docker_is_podman() {
    if ! command -v docker > /dev/null 2>&1; then
        return 1
    fi

    local docker_path podman_path docker_version docker_info

    # Same inode: `docker` is a symlink/hardlink to the podman binary.
    docker_path="$(type -P docker || true)"
    podman_path="$(type -P podman || true)"
    if [[ -n "${docker_path}" && -n "${podman_path}" && "${docker_path}" -ef "${podman_path}" ]]; then
        return 0
    fi

    # Authoritative signals that do not depend on the podman-docker wrapper's
    # "Emulate Docker CLI using podman" stderr message, which is suppressed when
    # /etc/containers/nodocker exists.
    docker_version="$(docker version --format '{{.Client.Name}}' 2> /dev/null || true)"
    if [[ "${docker_version}" == *[Pp]odman* ]]; then
        return 0
    fi

    docker_version="$(docker --version 2> /dev/null || true)"
    if [[ "${docker_version}" == *[Pp]odman* ]]; then
        return 0
    fi

    # Fallback for older podman-docker wrappers without /etc/containers/nodocker.
    docker_info="$(docker info 2>&1 || true)"
    [[ "${docker_info}" == *"Emulate Docker CLI using podman"* ]]
}

# Prefer docker when available, unless it is really Podman underneath.
detect_container_engine() {
    if command -v docker > /dev/null 2>&1 && container_engine_usable docker; then
        if docker_is_podman; then
            echo podman
        else
            echo docker
        fi
        return 0
    fi

    if command -v podman > /dev/null 2>&1 && container_engine_usable podman; then
        echo podman
        return 0
    fi

    echo "error: a working docker or podman daemon is required to run bash tests" >&2
    return 1
}

# Keep this destructive suite serial unless the caller explicitly opts in.
detect_jobs() {
    if [[ -n "${BASH_TEST_JOBS:-}" ]]; then
        echo "${BASH_TEST_JOBS}"
        return 0
    fi
    echo 1
}

# Force linux/amd64 only on macOS hosts that default to Apple Silicon containers.
detect_platform() {
    if [[ -n "${BASH_TEST_PLATFORM:-}" ]]; then
        echo "${BASH_TEST_PLATFORM}"
        return 0
    fi

    if [[ "$(uname -s)" == "Darwin" ]]; then
        echo "linux/amd64"
    fi
}

# Assemble engine-specific mounts and run ShellSpec inside the container.
main() {
    local engine jobs mount_arg report_dir report_mount
    engine="$(detect_container_engine)"
    jobs="$(detect_jobs)"
    report_dir="$(prepare_report_dir)" || return 1

    local -a engine_args=()
    if [[ "${engine}" == "podman" ]]; then
        mount_arg="${repo_root}:/work:Z"
        report_mount="${report_dir}:/work/.bash-test-report:Z"
        engine_args+=("--userns=keep-id:uid=0,gid=0")
    else
        mount_arg="${repo_root}:/work"
        report_mount="${report_dir}:/work/.bash-test-report"
    fi

    local -a platform_args=()
    local platform
    platform="$(detect_platform)"
    if [[ -n "${platform}" ]]; then
        platform_args=(--platform "${platform}")
    fi

    local junit_enabled="${BASH_TEST_JUNIT:-${CI:-0}}"
    local -a report_args=()
    if [[ "${junit_enabled}" == "true" || "${junit_enabled}" == "1" ]]; then
        report_args=(--output junit --reportdir /work/.bash-test-report)
        echo "ShellSpec JUnit report directory: ${report_dir}"
    fi

    echo "Running ShellSpec with ${engine} using image ${shellspec_image}"

    "${engine}" run --rm \
        "${platform_args[@]}" \
        "${engine_args[@]}" \
        -v "${mount_arg}" \
        -v "${report_mount}" \
        -w /work \
        "${shellspec_image}" \
        --jobs "${jobs}" \
        --format documentation \
        "${report_args[@]}" \
        "$@"
}

main "$@"
