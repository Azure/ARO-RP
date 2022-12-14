#!/bin/bash
#
# Perform actions to fix etcd members if the node's IP address doesn't match etcd endpoints and container status is not ready
# See for more information: https://docs.openshift.com/container-platform/4.4/backup_and_restore/replacing-unhealthy-etcd-member.html#restore-replace-crashlooping-etcd-member_replacing-unhealthy-etcd-member

main() {
    create_sym_links
    PATH+="${PATH}:/host/usr/bin"

    # ns="$1"

    host_pod="$2"
    local -i i=0
    local -A ips
    # Generate associative array of IPs
    for a in $@; do
        l="$(remove_single_quotes $a)"
        if jq -e . >/dev/null 2>&1 <<< "$l" && [[ $i -gt 1 ]] ; then
            d="$(jq -r '.[]' <<< "$l" | tr '\n' ' ')"
            IFS=' ' read -a arr <<< "$d"
            ips[${arr[0]}]="${arr[1]}"
        elif [[ $l =~ ^True|False$ ]] && [[ $i -gt 1 ]]; then
            container_status="$l"
            echo "Container status: $container_status"
        fi
        ((i++))
    done

    endpoints="${ips[ETCDCTL_ENDPOINTS]}"
    unset "ips[ETCDCTL_ENDPOINTS]"

    for k in "${!ips[@]}"; do
        if ! test_ips "${ips[$k]}" "$endpoints" ; then
            bad_node="$(get_name "node" $k)"
            bad_pod="$(get_name "pod" $k)"
            echo "bad node name is: $bad_node"
            echo "bad pod name is: $bad_pod"
            unset "ips[$k]"
        fi
    done
    echo "All good node IPs: ${ips[*]}"

    if [[ $container_status == "False" ]] && [[ "$bad_node" == "$K8S_NODE" ]]; then
        fix_host_etcd
        fix_peer_etcd
    fi
}

get_name() {
    s="$(tr '_' '-' <<< $2)"
    if [[ $1 == "pod" ]]; then
        s="${s//NODE/etcd}"
        echo "${s:0: -3}"
    elif [[ $1 == "node" ]]; then
        s="${s//NODE-/}"
        echo "${s//-IP/}"
    fi
}

remove_single_quotes() {
    local line
    line="$1"
    if [ "${line:0:1}" == \' ] || [ "${line: -1}" == \' ]; then
        line="$(echo $line | tr -d \')"
    fi
    echo "$line"
}

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

fix_host_etcd() {
    local bdir etcd_yaml
    bdir=/host/var/lib/etcd-backup
    etcd_yaml=/host/etc/kubernetes/manifests/etcd-pod.yaml
    etcd_dir=/host/var/lib/etcd/
    if [[ ! -d $bdir ]] && [[ -f $etcd_yaml ]]; then
        echo "Creating $bdir"
        mkdir -p "$bdir" || abort "failed to make backup directory"
        echo "Moving $etcd_yaml to $bdir"
        mv "$etcd_yaml" "$bdir/" || abort "failed to move $etcd_yaml to $bdir"
        echo "Moving $etcd_dir to /host/tmp"
        mv "$etcd_dir" /host/tmp/ || abort "failed to move $etcd_dir to /host/tmp"
    else
        echo "$bdir already exists or $etcd_yaml has already been moved"
        echo "Not taking host etcd backup"
    fi
}

########################################
# fix_peer_etcd removes the unhealthy member from healthy peer members lists
# and deletes unhealthy pod's secrets.
# Afterwards etcd is redeployed
########################################
fix_peer_etcd() {
    for p in ${!ips[@]}; do
        peer_pods+=("$(get_name "pod" "$p")")
    done

    # If at least one member was deleted from a peer's member list, delete secrets and patch
    cont="false"
    remove_peer_members
    if [[ $cont == "true" ]]; then
        remove_etcd_secrets
        oc patch etcd cluster -p='{"spec": {"forceRedeploymentReason": "single-master-recovery-'"$( date --rfc-3339=ns )"'"}}' --type=merge 
    fi
}

remove_etcd_secrets() {
    for pod in ${bad_pod//etcd-/etcd-peer-} ${bad_pod//etcd-/etcd-serving-} ${bad_pod//etcd-/etcd-serving-metrics-}; do
        oc delete secret -n openshift-etcd "$pod"
    done
}

########################################
# remove_peer_members remote shells into peer etcd members and removes unhealthy pod from their member's list
########################################
remove_peer_members() {
    declare -n deleted=cont
    for p in ${peer_pods[@]}; do
        members="$(oc rsh -n openshift-etcd -c etcdctl "pod/${p}" etcdctl member list -w json --hex true)" || abort "failed to list etcd members"
        id="$(jq -r --arg node "$bad_node" '.members[] | select( .name == $node).ID' <<< "$members")"
        if [[ -n $id ]]; then
            echo "rshing into pod/${p} now to remove member id $id"
            oc rsh \
                -n openshift-etcd \
                -c etcdctl \
                "pod/${p}" etcdctl member remove "$id"

            if [[ $? -eq 0 ]]; then
                deleted="true"
            fi
        else
            echo "$bad_node id not found in etcd member list for pod $p"
        fi
    done
}

test_ips() {
    local ip
    ip="$1"
    local endpoints
    endpoints="$2"
    if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        if grep -q "$ip" <<< "$endpoints"; then
            return 0
        else
            echo "$ip not found in ${endpoints}"
            return 1
        fi
    fi
}

abort() {
    echo "${1}, Aborting."
    exit 1
}

if [[ $EVENT == "Updated" ]]; then
    main "$@"
fi
