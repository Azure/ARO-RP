Describe 'bash unit test runner'
Include ./test/bash/spec/support/helpers.sh

run_runner_with_podman_docker() {
    ensure_test_workspace

    append_mock_command podman '
case "${1:-}" in
  info)
    if [ "${0##*/}" = "docker" ]; then
      printf "%s\n" "Emulate Docker CLI using podman. Create /etc/containers/nodocker to quiet msg." >&2
    fi
    exit 0
    ;;
  run)
    printf "%s %s\n" "${0##*/}" "$*"
    exit 0
    ;;
esac
'

    ln -s "${MOCK_BIN}/podman" "${MOCK_BIN}/docker"

    (
        PATH="${MOCK_BIN}:${ORIGINAL_PATH}" \
            BASH_TEST_REPORT_DIR="${TEST_ROOT}/report" \
            bash "${REPO_ROOT}/hack/unit-test-bash.sh"
    ) 2>&1
}

run_runner_with_silenced_podman_docker() {
    ensure_test_workspace

    # Mimic a podman-docker wrapper that is a *separate* script (not a symlink
    # to podman) and whose "Emulate Docker CLI using podman" info message is
    # silenced by /etc/containers/nodocker. Identity is only discoverable via
    # `docker version`/`docker --version`.
    append_mock_command podman '
case "${1:-}" in
  run)
    printf "%s %s\n" "${0##*/}" "$*"
    exit 0
    ;;
esac
'

    append_mock_command docker '
case "${1:-}" in
  info)
    exit 0
    ;;
  version)
    printf "%s\n" "Podman Engine"
    exit 0
    ;;
  --version)
    printf "%s\n" "podman version 5.8.3"
    exit 0
    ;;
esac
'

    (
        PATH="${MOCK_BIN}:${ORIGINAL_PATH}" \
            BASH_TEST_REPORT_DIR="${TEST_ROOT}/report" \
            bash "${REPO_ROOT}/hack/unit-test-bash.sh"
    ) 2>&1
}

run_runner_with_report_dir() {
    ensure_test_workspace

    append_mock_command docker '
case "${1:-}" in
  info)
    exit 0
    ;;
  run)
    printf "%s %s\n" "${0##*/}" "$*"
    exit 0
    ;;
esac
'

    (
        PATH="${MOCK_BIN}:${ORIGINAL_PATH}" \
            BASH_TEST_REPORT_DIR="$1" \
            bash "${REPO_ROOT}/hack/unit-test-bash.sh"
    ) 2>&1
}

run_runner_with_root_report_dir() {
    run_runner_with_report_dir "/"
}

run_runner_with_relative_report_dir() {
    run_runner_with_report_dir "."
}

It 'treats podman-docker as podman'
When call run_runner_with_podman_docker
The status should be success
The output should include 'Running ShellSpec with podman'
The output should include 'docker.io/shellspec/shellspec-debian:0.28.1'
The output should include 'podman run --rm'
The output should include '--userns=keep-id:uid=0,gid=0'
The output should include ':/work:Z'
End

It 'detects podman-docker when the emulate message is silenced'
When call run_runner_with_silenced_podman_docker
The status should be success
The output should include 'Running ShellSpec with podman'
The output should include 'podman run --rm'
The output should include '--userns=keep-id:uid=0,gid=0'
The output should include ':/work:Z'
End

It 'rejects root report directories'
When run run_runner_with_root_report_dir
The status should be failure
The output should include 'refusing unsafe report directory'
End

It 'rejects relative report directories'
When run run_runner_with_relative_report_dir
The status should be failure
The output should include 'refusing unsafe report directory'
End
End
