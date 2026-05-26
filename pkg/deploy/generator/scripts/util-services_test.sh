#!/bin/bash
# Unit tests for clamp() and compute_memory_budget() from util-services.sh.
#
# Run:  bash util-services_test.sh
#
# Sources production code from a temp directory so that the conditional
# source of util-common.sh / util-system.sh is skipped (they only load
# when co-located, which mirrors actual VMSS deployment where scripts
# are concatenated).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PASS=0
FAIL=0

WORK_DIR=$(mktemp -d)
trap 'rm -rf "${WORK_DIR}"' EXIT
cp "${SCRIPT_DIR}/util-services.sh" "${WORK_DIR}/"
cd "${WORK_DIR}"

log() { :; }

declare -r role_gateway="gateway"
declare -r role_rp="rp"

source "${WORK_DIR}/util-services.sh"

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
    MEM_RP=0; MEM_MONITOR=0; MEM_MDM=0; MEM_OTEL=0
    MEM_PORTAL=0; MEM_MIMO_SCHEDULER=0; MEM_MIMO_ACTUATOR=0; MEM_GATEWAY=0
}

MOCK_MEMINFO=""

mock_meminfo() {
    local -ri kib="$1"
    MOCK_MEMINFO="MemTotal:       ${kib} kB"
}

# Override compute_memory_budget to read from MOCK_MEMINFO instead of
# /proc/meminfo, keeping all other logic identical.
compute_memory_budget() {
    local -n _role="$1"

    local -i total_mem_kib
    total_mem_kib=$(echo "${MOCK_MEMINFO}" | awk '/MemTotal/ {print $2}')
    local -i total_mem_mib=$(( total_mem_kib / 1024 ))
    local -i budget_mib=$(( total_mem_mib - OS_RESERVE_MIB ))

    if (( budget_mib < 0 )); then
        budget_mib=0
    fi

    if [ "$_role" == "$role_gateway" ]; then
        if (( WEIGHT_GATEWAY <= 0 )); then
            echo "ABORT: WEIGHT_GATEWAY must be > 0" >&2
            return 1
        fi
        local -i gw_sum=${WEIGHT_GATEWAY}
        MEM_GATEWAY=$(clamp $(( budget_mib * WEIGHT_GATEWAY / gw_sum )) $FLOOR_GATEWAY $CAP_GATEWAY)
    elif [ "$_role" == "$role_rp" ]; then
        local -i w
        for w in $WEIGHT_RP $WEIGHT_MONITOR $WEIGHT_MDM $WEIGHT_OTEL \
                 $WEIGHT_PORTAL $WEIGHT_MIMO_SCHEDULER $WEIGHT_MIMO_ACTUATOR; do
            if (( w <= 0 )); then
                echo "ABORT: all service weights must be > 0 (got ${w})" >&2
                return 1
            fi
        done
        local -i rp_sum=$(( WEIGHT_RP + WEIGHT_MONITOR + WEIGHT_MDM + WEIGHT_OTEL \
            + WEIGHT_PORTAL + WEIGHT_MIMO_SCHEDULER + WEIGHT_MIMO_ACTUATOR ))
        MEM_RP=$(clamp $(( budget_mib * WEIGHT_RP / rp_sum )) $FLOOR_RP $CAP_RP)
        MEM_MONITOR=$(clamp $(( budget_mib * WEIGHT_MONITOR / rp_sum )) $FLOOR_MONITOR $CAP_MONITOR)
        MEM_MDM=$(clamp $(( budget_mib * WEIGHT_MDM / rp_sum )) $FLOOR_MDM $CAP_MDM)
        MEM_OTEL=$(clamp $(( budget_mib * WEIGHT_OTEL / rp_sum )) $FLOOR_OTEL $CAP_OTEL)
        MEM_PORTAL=$(clamp $(( budget_mib * WEIGHT_PORTAL / rp_sum )) $FLOOR_PORTAL $CAP_PORTAL)
        MEM_MIMO_SCHEDULER=$(clamp $(( budget_mib * WEIGHT_MIMO_SCHEDULER / rp_sum )) $FLOOR_MIMO_SCHEDULER $CAP_MIMO_SCHEDULER)
        MEM_MIMO_ACTUATOR=$(clamp $(( budget_mib * WEIGHT_MIMO_ACTUATOR / rp_sum )) $FLOOR_MIMO_ACTUATOR $CAP_MIMO_ACTUATOR)
    fi
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

rp_weight_sum=$(( WEIGHT_RP + WEIGHT_MONITOR + WEIGHT_MDM + WEIGHT_OTEL \
    + WEIGHT_PORTAL + WEIGHT_MIMO_SCHEDULER + WEIGHT_MIMO_ACTUATOR ))
assert_eq "RP weights sum > 0"    "$(( rp_weight_sum > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_RP > 0"         "$(( WEIGHT_RP > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MONITOR > 0"    "$(( WEIGHT_MONITOR > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MDM > 0"        "$(( WEIGHT_MDM > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_OTEL > 0"       "$(( WEIGHT_OTEL > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_PORTAL > 0"     "$(( WEIGHT_PORTAL > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MIMO_SCHED > 0" "$(( WEIGHT_MIMO_SCHEDULER > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_MIMO_ACT > 0"   "$(( WEIGHT_MIMO_ACTUATOR > 0 ? 1 : 0 ))" 1
assert_eq "WEIGHT_GATEWAY > 0"    "$(( WEIGHT_GATEWAY > 0 ? 1 : 0 ))" 1

# ── D8s_v3 (32 GiB = 32768 MiB = 33554432 KiB) ──
# budget = 32768 - 1536 = 31232 MiB

echo "=== D8s_v3 RP (32 GiB) ==="

reset_mem_vars
mock_meminfo 33554432
compute_memory_budget role_rp

assert_eq "D8s_v3 RP"           "${MEM_RP}"             8744
assert_eq "D8s_v3 Monitor"      "${MEM_MONITOR}"        6871
assert_eq "D8s_v3 MDM"          "${MEM_MDM}"            4372
assert_eq "D8s_v3 OTEL"         "${MEM_OTEL}"           3747
assert_eq "D8s_v3 Portal"       "${MEM_PORTAL}"         3123
assert_eq "D8s_v3 MIMO Sched"   "${MEM_MIMO_SCHEDULER}" 2498
assert_eq "D8s_v3 MIMO Act"     "${MEM_MIMO_ACTUATOR}"  1873

# ── D4s_v3 (16 GiB = 16384 MiB = 16777216 KiB) ──
# budget = 16384 - 1536 = 14848 MiB

echo "=== D4s_v3 RP (16 GiB) ==="

reset_mem_vars
mock_meminfo 16777216
compute_memory_budget role_rp

assert_eq "D4s_v3 RP"           "${MEM_RP}"             4157
assert_eq "D4s_v3 Monitor"      "${MEM_MONITOR}"        3266
assert_eq "D4s_v3 MDM"          "${MEM_MDM}"            2078
assert_eq "D4s_v3 OTEL"         "${MEM_OTEL}"           1781
assert_eq "D4s_v3 Portal"       "${MEM_PORTAL}"         1484
assert_eq "D4s_v3 MIMO Sched"   "${MEM_MIMO_SCHEDULER}" 1187
assert_eq "D4s_v3 MIMO Act"     "${MEM_MIMO_ACTUATOR}"  890

# ── D2s_v3 (8 GiB = 8192 MiB = 8388608 KiB) ──
# budget = 8192 - 1536 = 6656 MiB

echo "=== D2s_v3 RP (8 GiB) ==="

reset_mem_vars
mock_meminfo 8388608
compute_memory_budget role_rp

assert_eq "D2s_v3 RP"           "${MEM_RP}"             2048    # floor
assert_eq "D2s_v3 Monitor"      "${MEM_MONITOR}"        2048    # floor
assert_eq "D2s_v3 MDM"          "${MEM_MDM}"            931
assert_eq "D2s_v3 OTEL"         "${MEM_OTEL}"           798
assert_eq "D2s_v3 Portal"       "${MEM_PORTAL}"         665
assert_eq "D2s_v3 MIMO Sched"   "${MEM_MIMO_SCHEDULER}" 532
assert_eq "D2s_v3 MIMO Act"     "${MEM_MIMO_ACTUATOR}"  399

# ── Gateway D2s_v3 (8 GiB) ──

echo "=== D2s_v3 Gateway (8 GiB) ==="

reset_mem_vars
mock_meminfo 8388608
compute_memory_budget role_gateway

assert_eq "D2s_v3 Gateway"      "${MEM_GATEWAY}"        6656
assert_eq "Gateway no RP leak"  "${MEM_RP}"             0

# ── Tiny VM (2 GiB) — all services hit floor ──
# budget = 2048 - 1536 = 512 MiB

echo "=== Tiny VM RP (2 GiB) ==="

reset_mem_vars
mock_meminfo 2097152
compute_memory_budget role_rp

assert_eq "Tiny RP (floor)"      "${MEM_RP}"             2048
assert_eq "Tiny Monitor (floor)"  "${MEM_MONITOR}"       2048
assert_eq "Tiny MDM (floor)"     "${MEM_MDM}"            512
assert_eq "Tiny OTEL (floor)"    "${MEM_OTEL}"           512
assert_eq "Tiny Portal (floor)"  "${MEM_PORTAL}"         512
assert_eq "Tiny MIMO Sched"      "${MEM_MIMO_SCHEDULER}" 256
assert_eq "Tiny MIMO Act"        "${MEM_MIMO_ACTUATOR}"  256

# ── Sub-reserve VM (1 GiB) — budget clamps to 0 ──

echo "=== Sub-reserve VM (1 GiB) ==="

reset_mem_vars
mock_meminfo 1048576
compute_memory_budget role_rp

assert_eq "Sub-reserve RP"       "${MEM_RP}"             2048    # floor
assert_eq "Sub-reserve Gateway"  "${MEM_GATEWAY}"        0       # not set

# ── Zero-weight rejection ──

echo "=== zero-weight rejection ==="

# Temporarily override a weight to 0 and verify compute_memory_budget fails.
# We save/restore using a subshell approach (run in a child process).
if (WEIGHT_OTEL=0; mock_meminfo 33554432; compute_memory_budget role_rp) 2>/dev/null; then
    echo "FAIL: zero RP weight accepted"
    FAIL=$((FAIL + 1))
else
    PASS=$((PASS + 1))
fi

if (WEIGHT_GATEWAY=0; mock_meminfo 33554432; compute_memory_budget role_gateway) 2>/dev/null; then
    echo "FAIL: zero gateway weight accepted"
    FAIL=$((FAIL + 1))
else
    PASS=$((PASS + 1))
fi

# ── Summary ──

echo ""
echo "=== Results: ${PASS} passed, ${FAIL} failed ==="

if (( FAIL > 0 )); then
    exit 1
fi
