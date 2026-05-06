set -eu

is_container_test_environment() {
  [ "${container:-}" = "docker" ] ||
    [ "${container:-}" = "podman" ] ||
    [ -f "/.dockerenv" ] ||
    [ -f "/run/.containerenv" ]
}

spec_helper_precheck() {
  minimum_version "0.28.0"
  if [ "$SHELL_TYPE" != "bash" ]; then
    abort "Only bash is supported."
  fi

  if [ "${BASH_TEST_ALLOW_HOST:-0}" != "1" ] && ! is_container_test_environment; then
    abort "Refusing to run destructive bash tests on the host. Run inside a container or set BASH_TEST_ALLOW_HOST=1 to opt in."
  fi
}

spec_helper_loaded() {
  # shellcheck source=test/bash/spec/support/helpers.sh
  . "${SHELLSPEC_HELPERDIR}/support/helpers.sh"
}

spec_helper_configure() {
  before_each 'setup_test_workspace'
  after_each 'cleanup_test_workspace'
}

setup_test_workspace() {
  :
}

cleanup_test_workspace() {
  reset_absolute_test_state
  rm -f /etc/ssh/sshd_config
  if [ "${TEST_ROOT:-}" ]; then
    case "$TEST_ROOT" in
      "${TMPDIR:-/tmp}"/aro-bash-tests.*)
        rm -rf -- "$TEST_ROOT"
        ;;
    esac
  fi
}
