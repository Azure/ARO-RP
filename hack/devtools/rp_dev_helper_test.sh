#!/bin/bash
# Run from the repository root: bash hack/devtools/rp_dev_helper_test.sh

set -euo pipefail

source hack/devtools/rp_dev_helper.sh

digest=sha256:2597e3942ecda57062eed0171d2faf125ef6373772660cab5a175e185b10ba92
repository=oss/otel/opentelemetry-collector-contrib
import_status=0
show_status=0
show_digest="$digest"
import_count=0
import_args=()

# All Azure operations are mocked; these tests never contact a registry.
az() {
    if [[ "${1-} ${2-}" == "acr import" ]]; then
        import_args=("$@")
        import_count=$((import_count + 1))
        return "$import_status"
    fi
    if [[ "$*" == "acr repository show --name devacr --image $repository@$digest --query digest --output tsv --only-show-errors" ]]; then
        printf '%s\n' "$show_digest"
        return "$show_status"
    fi
    echo "Unexpected Azure command: $*" >&2
    return 99
}

expect_failure() {
    local expected="$1"
    shift
    local output
    if output=$("$@" 2>&1); then
        echo "Expected failure: $*" >&2
        exit 1
    fi
    if [[ "$output" != *"$expected"* ]]; then
        echo "Missing expected error '$expected' in: $output" >&2
        exit 1
    fi
}

mirror_mise_otel_image devacr
expected_args="acr import --name devacr --source mcr.microsoft.com/$repository@$digest --image $repository:${digest#sha256:} --force --only-show-errors --output none"
if [[ "${import_args[*]}" != "$expected_args" ]]; then
    echo "Unexpected import arguments: ${import_args[*]}" >&2
    exit 1
fi
mirror_mise_otel_image devacr
[[ "$import_count" -eq 2 ]]

expect_failure "Usage:" mirror_mise_otel_image
expect_failure "Usage:" mirror_mise_otel_image devacr.azurecr.io
expect_failure "Usage:" mirror_mise_otel_image devacr extra

import_status=7
expect_failure "Failed to import" mirror_mise_otel_image devacr
import_status=0
show_status=8
expect_failure "Unable to verify" mirror_mise_otel_image devacr
show_status=0
show_digest=sha256:incorrect
expect_failure "digest mismatch" mirror_mise_otel_image devacr
show_digest="$digest"

VERSION_CONST_FILE=/dev/null expect_failure "Expected a single digest-pinned" mirror_mise_otel_image devacr
VERSION_CONST_FILE=/does-not-exist expect_failure "Unable to read" mirror_mise_otel_image devacr
VERSION_CONST_FILE=/dev/stdin expect_failure "Expected a single digest-pinned" mirror_mise_otel_image devacr <<'EOF'
func OTelImage(acrDomain string) string {
    return acrDomain + "/oss/otel/opentelemetry-collector-contrib:0.95.0-linux-amd64"
}
EOF

# A new central pin must be used without changing the helper.
digest=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
show_digest="$digest"
VERSION_CONST_FILE=/dev/stdin mirror_mise_otel_image devacr <<EOF
func OTelImage(acrDomain string) string {
    return acrDomain + "/$repository@$digest"
}
EOF
if [[ "${import_args[*]}" != "acr import --name devacr --source mcr.microsoft.com/$repository@$digest --image $repository:${digest#sha256:} --force --only-show-errors --output none" ]]; then
    echo "Import did not use the image from the central version definition" >&2
    exit 1
fi

echo "MISE OTEL developer preparation tests passed"
