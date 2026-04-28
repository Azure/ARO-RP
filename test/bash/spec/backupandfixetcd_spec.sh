Describe 'backupandfixetcd.sh'
  Include ./test/bash/spec/support/helpers.sh
  script_path="${REPO_ROOT}/pkg/frontend/scripts/backupandfixetcd.sh"

  run_backup_mode() {
    ensure_test_workspace
    mkdir -p /etc/kubernetes/manifests /var/lib/etcd
    printf 'manifest-data' > /etc/kubernetes/manifests/etcd-pod.yaml
    printf 'etcd-data' > /var/lib/etcd/member

    BACKUP=1 FIX_PEERS='' bash "${script_path}"

    backup_dir="$(echo /var/lib/etcd-backup-*)"
    printf '%s\n' "${backup_dir}"
    printf '%s\n' "$(cat "${backup_dir}/etcd-pod.yaml")"
    printf '%s\n' "$(cat "${backup_dir}/etcd/member")"
    if backup_paths_moved; then
      printf '%s\n' 'moved=yes'
    else
      printf '%s\n' 'moved=no'
    fi
  }

  backup_paths_moved() {
    [ ! -e /var/lib/etcd ] && [ ! -e /etc/kubernetes/manifests/etcd-pod.yaml ]
  }

  run_fix_peers_mode() {
    ensure_test_workspace

    append_mock_command oc '
      printf "oc %s\n" "$*" >> "${CALL_LOG}"
      if [[ "$*" == *"member list"* ]]; then
        printf "%s\n" "{\"members\":[{\"name\":\"master-0\",\"ID\":\"member-123\"}]}"
      fi
    '
    append_mock_command jq 'printf "%s\n" "member-123"'
    cp "${MOCK_BIN}/oc" /host/usr/bin/oc
    cp "${MOCK_BIN}/jq" /host/usr/bin/jq

    FIX_PEERS=1 \
      BACKUP='' \
      PEER_PODS="etcd-0 etcd-1" \
      DEGRADED_NODE="master-0" \
      bash "${script_path}"

    cat "${CALL_LOG}"
  }

  It 'backs up etcd manifests and data into a timestamped directory'
    When call run_backup_mode
    The status should be success
    The output should include '/var/lib/etcd-backup-'
    The output should include 'manifest-data'
    The output should include 'etcd-data'
    The output should include 'moved=yes'
  End

  It 'removes degraded peer members when FIX_PEERS is set'
    When call run_fix_peers_mode
    The status should be success
    The output should include 'Starting peer etcd member removal'
    The output should include 'oc rsh -n openshift-etcd -c etcdctl pod/etcd-0 etcdctl member remove member-123'
    The output should include 'oc rsh -n openshift-etcd -c etcdctl pod/etcd-1 etcdctl member remove member-123'
  End
End
