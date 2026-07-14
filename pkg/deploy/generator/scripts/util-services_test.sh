#!/bin/bash
# Unit tests for clamp() and compute_memory_budget() from util-services.sh.
#
# Run:  bash util-services_test.sh
#
# Overrides only host_mem_mib() so all allocation logic is tested
# against the real production compute_memory_budget() function.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PASS=0
FAIL=0

WORK_DIR=$(mktemp -d)
trap 'rm -rf "${WORK_DIR}"' EXIT
cp "${SCRIPT_DIR}/util-services.sh" "${WORK_DIR}/"
cd "${WORK_DIR}"

log() { :; }
abort() { echo "ABORT: $*" >&2; return 1; }

declare -r role_gateway="gateway"
declare -r role_rp="rp"

source "${WORK_DIR}/util-services.sh"

# Override only the meminfo source — all other logic is the real
# production code from util-services.sh.
MOCK_HOST_MEM_MIB=0
host_mem_mib() {
    echo "${MOCK_HOST_MEM_MIB}"
}

mock_meminfo() {
    MOCK_HOST_MEM_MIB="$1"
}

assert_eq() {
    local -r label="$1"
    local -ri got="$2"
    local -ri want="$3"
    if (( got == want )); then
        PASS=$((PASS + 1))
    else
        echo "FAIL: ${label}: got ${got}, want ${want}"
        FAIL=$((FAIL + 1))
    fi
}

reset_mem_vars() {
    MEM_RP=0; MEM_MONITOR=0; MEM_OTEL=0
    MEM_PORTAL=0; MEM_MIMO_SCHEDULER=0; MEM_MIMO_ACTUATOR=0
}

# ── clamp() tests ──

echo "=== clamp() ==="

assert_eq "value within range"    "$(clamp 1000 512 2048)" 1000
assert_eq "value below floor"     "$(clamp 100 512 2048)"  512
assert_eq "value above cap"       "$(clamp 5000 512 2048)" 2048
assert_eq "value at floor"        "$(clamp 512 512 2048)"  512
assert_eq "value at cap"          "$(clamp 2048 512 2048)" 2048
assert_eq "cap=0 (uncapped)"      "$(clamp 9999 512 0)"    9999
assert_eq "cap=0 below floor"     "$(clamp 100 512 0)"     512
assert_eq "zero value with floor" "$(clamp 0 256 0)"       256
assert_eq "negative value"        "$(clamp -100 256 4096)" 256

# ── Weight sanity ──

echo "=== weight sanity ==="

rp_weight_sum=$(( WEIGHT_RP + WEIGHT_MONITOR + WEIGHT_OTEL \
    + WEIGHT_PORTAL + WEIGHT_MIMO_SCHEDULER + WEIGHT_MIMO_ACTUATOR ))
assert_eq "RP weights sum > 0"    "$(( rp_weight_sum > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_RP > 0"         "$(( WEIGHT_RP > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MONITOR > 0"    "$(( WEIGHT_MONITOR > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_OTEL > 0"       "$(( WEIGHT_OTEL > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_PORTAL > 0"     "$(( WEIGHT_PORTAL > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MIMO_SCHED > 0" "$(( WEIGHT_MIMO_SCHEDULER > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MIMO_ACT > 0"   "$(( WEIGHT_MIMO_ACTUATOR > 0 ? 1 : 0 ))" 1

# ── D8s_v3 (32 GiB = 32768 MiB) ──
# budget = 32768 - 1536 = 31232 MiB
# weight_sum = 28+22+12+10+8+6 = 86

echo "=== D8s_v3 RP (32 GiB) ==="

reset_mem_vars
mock_meminfo 32768
compute_memory_budget

assert_eq "D8s_v3 RP"           "${MEM_RP}"             10168
assert_eq "D8s_v3 Monitor"      "${MEM_MONITOR}"        7989
assert_eq "D8s_v3 OTEL"         "${MEM_OTEL}"           4096   # cap
assert_eq "D8s_v3 Portal"       "${MEM_PORTAL}"         3631
assert_eq "D8s_v3 MIMO Sched"   "${MEM_MIMO_SCHEDULER}" 2905
assert_eq "D8s_v3 MIMO Act"     "${MEM_MIMO_ACTUATOR}"  2048   # cap

# ── D4s_v3 (16 GiB = 16384 MiB) ──
# budget = 16384 - 1536 = 14848 MiB

echo "=== D4s_v3 RP (16 GiB) ==="

reset_mem_vars
mock_meminfo 16384
compute_memory_budget

assert_eq "D4s_v3 RP"           "${MEM_RP}"             4834
assert_eq "D4s_v3 Monitor"      "${MEM_MONITOR}"        3798
assert_eq "D4s_v3 OTEL"         "${MEM_OTEL}"           2071
assert_eq "D4s_v3 Portal"       "${MEM_PORTAL}"         1726
assert_eq "D4s_v3 MIMO Sched"   "${MEM_MIMO_SCHEDULER}" 1381
assert_eq "D4s_v3 MIMO Act"     "${MEM_MIMO_ACTUATOR}"  1035

# ── D2s_v3 (8 GiB = 8192 MiB) ──
# budget = 8192 - 1536 = 6656 MiB

echo "=== D2s_v3 RP (8 GiB) ==="

reset_mem_vars
mock_meminfo 8192
compute_memory_budget

assert_eq "D2s_v3 RP"           "${MEM_RP}"             2167
assert_eq "D2s_v3 Monitor"      "${MEM_MONITOR}"        2048   # floor
assert_eq "D2s_v3 OTEL"         "${MEM_OTEL}"           928
assert_eq "D2s_v3 Portal"       "${MEM_PORTAL}"         773
assert_eq "D2s_v3 MIMO Sched"   "${MEM_MIMO_SCHEDULER}" 619
assert_eq "D2s_v3 MIMO Act"     "${MEM_MIMO_ACTUATOR}"  464

# ── Tiny VM (2 GiB) — all services hit floor ──
# budget = 2048 - 1536 = 512 MiB

echo "=== Tiny VM RP (2 GiB) ==="

reset_mem_vars
mock_meminfo 2048
compute_memory_budget

assert_eq "Tiny RP (floor)"      "${MEM_RP}"             2048
assert_eq "Tiny Monitor (floor)" "${MEM_MONITOR}"        2048
assert_eq "Tiny OTEL (floor)"    "${MEM_OTEL}"           512
assert_eq "Tiny Portal (floor)"  "${MEM_PORTAL}"         512
assert_eq "Tiny MIMO Sched"      "${MEM_MIMO_SCHEDULER}" 256
assert_eq "Tiny MIMO Act"        "${MEM_MIMO_ACTUATOR}"  256

# ── Sub-reserve VM (1 GiB) — budget clamps to 0 ──

echo "=== Sub-reserve VM (1 GiB) ==="

reset_mem_vars
mock_meminfo 1024
compute_memory_budget

assert_eq "Sub-reserve RP"       "${MEM_RP}"             2048   # floor
assert_eq "Sub-reserve Monitor"  "${MEM_MONITOR}"        2048   # floor

# ── Zero-weight rejection ──
# The weights are declared -ir (readonly) in util-services.sh, so we
# cannot override them in-process. Instead, we create a modified copy
# of util-services.sh with the target weight set to 0 and source it
# in a subshell.

echo "=== zero-weight rejection ==="

test_zero_weight() {
    local -r var_name="$1"
    local -r label="$2"

    local modified="${WORK_DIR}/zero-weight-test.sh"
    sed "s/declare -ir ${var_name}=[0-9]*/declare -ir ${var_name}=0/" \
        "${WORK_DIR}/util-services.sh" > "${modified}"

    if bash -c "
        set -euo pipefail
        log() { :; }
        abort() { echo \"ABORT: \$*\" >&2; return 1; }
        declare -r role_gateway='gateway'
        declare -r role_rp='rp'
        source '${modified}'
        host_mem_mib() { echo 32768; }
        compute_memory_budget
    " 2>/dev/null; then
        echo "FAIL: ${label}"
        FAIL=$((FAIL + 1))
    else
        PASS=$((PASS + 1))
    fi
    rm -f "${modified}"
}

test_zero_weight "WEIGHT_OTEL"           "zero OTEL weight accepted"
test_zero_weight "WEIGHT_RP"             "zero RP weight accepted"
test_zero_weight "WEIGHT_MONITOR"        "zero Monitor weight accepted"
test_zero_weight "WEIGHT_PORTAL"         "zero Portal weight accepted"
test_zero_weight "WEIGHT_MIMO_SCHEDULER" "zero MIMO Sched weight accepted"
test_zero_weight "WEIGHT_MIMO_ACTUATOR"  "zero MIMO Act weight accepted"

# ── Summary ──

echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if (( FAIL > 0 )); then
    exit 1
fi
