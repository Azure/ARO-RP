#!/bin/bash
# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.
#
# Prints the top 10 namespaces by etcd key count, plus a <cluster-scope> bucket.
# Fetches the namespace list from etcd to unambiguously classify NF==5 keys:
#   /kubernetes.io/RESOURCE/NAMESPACE/NAME        (NF==5, $4 is a namespace)
#   /kubernetes.io/GROUP/RESOURCE/NAMESPACE/NAME  (NF==6, $5 is a namespace)
#   everything else → <cluster-scope>
namespaces=$(etcdctl get --prefix /kubernetes.io/namespaces/ --keys-only \
  | awk -F'/' 'NF==4 {print $4}' | tr '\n' ',')
etcdctl get --prefix /kubernetes.io/ --keys-only \
  | awk -F'/' -v ns_csv="$namespaces" '
    BEGIN { n = split(ns_csv, a, ","); for (i=1; i<=n; i++) ns[a[i]] = 1 }
    /^\/kubernetes.io\// {
      if      (NF==5 && ($4 in ns)) count[$4]++
      else if (NF==6)               count[$5]++
      else                          count["<cluster-scope>"]++
    }
    END { for (k in count) print count[k], k }' \
  | sort -nr \
  | head -n 10
