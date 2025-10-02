#!/bin/bash
#
# See for more information: https://docs.openshift.com/container-platform/4.10/backup_and_restore/control_plane_backup_and_restore/replacing-unhealthy-etcd-member.html

remove_peer_members() {
  echo "${PEER_PODS}"
	for p in ${PEER_PODS}; do
		echo "Attempting to get ID for pod/${p}"
		members="$(oc rsh -n openshift-etcd -c etcdctl "pod/${p}" etcdctl member list -w json --hex true)"
		id="$(jq -r --arg node "$DEGRADED_NODE" '.members[] | select( .name == $node).ID' <<< "$members")"
		echo "id: ${id:-Not Found}"
		if [[ -n $id ]]; then
			echo "rshing into pod/${p} now to remove member id $id"
			oc rsh \
				-n openshift-etcd \
				-c etcdctl \
				"pod/${p}" etcdctl member remove "$id"
		else
			echo "${DEGRADED_NODE} id not found in etcd member list for pod ${p}"
		fi
	done
}

# jq expects it's required shared libraries to be present in /usr/lib64, not /host/usr/lib64.
# Because we are using jq mount under /host and haven't installed jq, those libraries exist under /host/usr/lib64 rather than /usr/lib64.
# Creating the symbolic links allows jq to resolve it's libraries without the need for installing.
create_sym_links() {
    jq_lib1="/usr/lib64/libjq.so.1"
    jq_lib2="/usr/lib64/libonig.so.5"

    if [[ ! -f $jq_lib1 ]]; then
        ln -s "/host${jq_lib1}" "$jq_lib1"
    fi
    if [[ ! -f $jq_lib2 ]]; then
        ln -s "/host${jq_lib2}" "$jq_lib2"
    fi
}

backup_etcd() {
    local bdir etcd_yaml etcd_dir
    bdir="/var/lib/etcd-backup-$(date +%Y%m%d%H%M%S)"
    etcd_yaml=/etc/kubernetes/manifests/etcd-pod.yaml
    etcd_dir=/var/lib/etcd
    if [[ -d $etcd_dir ]] && [[ -f $etcd_yaml ]]; then
        echo "Creating $bdir"
        mkdir -p "$bdir" || abort "failed to make backup directory"
        echo "Moving $etcd_yaml to $bdir"
        mv "$etcd_yaml" "$bdir" || abort "failed to move $etcd_yaml to $bdir"
        echo "Moving $etcd_dir to /host/tmp"
        mv "$etcd_dir" $bdir || abort "failed to move $etcd_dir to $bdir"
    else
        echo "$etcd_dir doesn't exist or $etcd_yaml has already been moved"
        echo "Not taking host etcd backup"
    fi
}

abort() {
    echo "${1}, Aborting."
    exit 1
}

if [[ -n $FIX_PEERS ]]; then
    PATH+="${PATH}:/host/usr/bin"
    create_sym_links
    echo "Starting peer etcd member removal"
    remove_peer_members
elif [[ -n $BACKUP ]]; then
    echo "Starting etcd data backup"
    backup_etcd
else
    abort "BACKUP and FIX_PEERS are unset, no actions taken."
fi
