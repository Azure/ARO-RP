#!/bin/bash
# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.
#
# Profile the ARO RP under load for specific endpoints.
# Usage:
#   ./hack/pprof-profile-endpoint.sh ENDPOINT=/api/v1/clusters DURATION=20s RATE=100
#   ./hack/pprof-profile-endpoint.sh ENDPOINT=all DURATION=10s RATE=50

set -euo pipefail

# Default values
ENDPOINT="${ENDPOINT:-}"
DURATION="${DURATION:-20s}"
RATE="${RATE:-100}"
PPROF_HOST="${PPROF_HOST:-127.0.0.1}"
PPROF_PORT="${PPROF_PORT:-6060}"
PPROF_URL="http://${PPROF_HOST}:${PPROF_PORT}"
PPROF_OUTPUT_DIR="${PPROF_OUTPUT_DIR:-./pprof-data}"
LOADTEST_BASE_URL="${LOADTEST_BASE_URL:-https://localhost:8443}"

# Test values for path parameters
TEST_SUBSCRIPTION_ID="${TEST_SUBSCRIPTION_ID:-00000000-0000-0000-0000-000000000000}"
TEST_RESOURCE_GROUP="${TEST_RESOURCE_GROUP:-test-rg}"
TEST_LOCATION="${TEST_LOCATION:-eastus}"
TEST_RESOURCE_NAME="${TEST_RESOURCE_NAME:-test-cluster}"
TEST_OPENSHIFT_VERSION="${TEST_OPENSHIFT_VERSION:-4.14.0}"
TEST_OPENSHIFT_MINOR_VERSION="${TEST_OPENSHIFT_MINOR_VERSION:-4.14}"
TEST_OPERATION_ID="${TEST_OPERATION_ID:-00000000-0000-0000-0000-000000000000}"
TEST_DETECTOR_ID="${TEST_DETECTOR_ID:-test-detector}"
TEST_SYNC_SET_NAME="${TEST_SYNC_SET_NAME:-test-syncset}"
TEST_MANIFEST_ID="${TEST_MANIFEST_ID:-test-manifest}"
TEST_DEPLOYMENT_NAME="${TEST_DEPLOYMENT_NAME:-test-deployment}"

# API version to use for requests (default to latest stable)
# Valid versions: 2020-04-30, 2022-04-01, 2022-09-04, 2023-04-01, 2023-09-04, 2023-11-22, 2025-07-25
# Preview versions: 2021-09-01-preview, 2023-07-01-preview, 2024-08-12-preview
# Admin version: admin
TEST_API_VERSION="${TEST_API_VERSION:-2025-07-25}"

# Find the latest swagger file
SWAGGER_DIR="swagger/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/openshiftclusters"
LATEST_SWAGGER=$(find "${SWAGGER_DIR}" -name "redhatopenshift.json" -type f | sort -V | tail -1)

if [ -z "$LATEST_SWAGGER" ]; then
    echo "Error: Could not find swagger file in ${SWAGGER_DIR}"
    exit 1
fi

# Sanitize endpoint name for filesystem
sanitize_endpoint() {
    local result
    result=$(sed 's|^/||; s|/|-|g; s|[^a-zA-Z0-9-]|_|g' <<< "$1")
    tr '[:upper:]' '[:lower:]' <<< "$result" | sed 's|__*|_|g'
}

# Replace path parameters with test values
substitute_path_params() {
    local path="$1"
    
    # Replace each parameter placeholder with its test value
    path=$(sed "s|{subscriptionId}|${TEST_SUBSCRIPTION_ID}|g" <<< "$path")
    path=$(sed "s|{resourceGroupName}|${TEST_RESOURCE_GROUP}|g" <<< "$path")
    path=$(sed "s|{resourceProviderNamespace}|Microsoft.RedHatOpenShift|g" <<< "$path")
    path=$(sed "s|{resourceType}|openShiftClusters|g" <<< "$path")
    path=$(sed "s|{resourceName}|${TEST_RESOURCE_NAME}|g" <<< "$path")
    path=$(sed "s|{location}|${TEST_LOCATION}|g" <<< "$path")
    path=$(sed "s|{openShiftVersion}|${TEST_OPENSHIFT_VERSION}|g" <<< "$path")
    path=$(sed "s|{openShiftMinorVersion}|${TEST_OPENSHIFT_MINOR_VERSION}|g" <<< "$path")
    path=$(sed "s|{operationId}|${TEST_OPERATION_ID}|g" <<< "$path")
    path=$(sed "s|{detectorId}|${TEST_DETECTOR_ID}|g" <<< "$path")
    path=$(sed "s|{syncsetname}|${TEST_SYNC_SET_NAME}|g" <<< "$path")
    path=$(sed "s|{manifestId}|${TEST_MANIFEST_ID}|g" <<< "$path")
    path=$(sed "s|{deploymentName}|${TEST_DEPLOYMENT_NAME}|g" <<< "$path")
    
    # Remove any remaining unmatched {param} patterns and stray } characters
    sed 's|{[^}]*}||g; s|}||g' <<< "$path"
}

# Extract endpoints from swagger file
# Returns: path<tab>method (e.g., "/path/to/endpoint<tab>GET")
extract_endpoints_from_swagger() {
    local swagger_file="$1"
    
    command -v jq >/dev/null 2>&1 || {
        echo "Error: jq is required to parse swagger files. Install with: brew install jq (macOS) or apt-get install jq (Linux)"
        exit 1
    }
    
    # Extract paths with their HTTP methods (format: path<tab>method)
    {
        jq -r '.paths | to_entries[] | 
            .key as $path | 
            .value | 
            to_entries[] | 
            select(.key | test("^(get|post|put|patch|delete)$"; "i")) | 
            "\($path)\t\(.key)"' "$swagger_file" | \
            awk '{print $1 "\t" toupper($2)}' | \
            grep -v "^/admin" || true
        printf '/healthz/ready\tGET\n'
    } | sort -u
}

# Profile a single endpoint
# Parameters: endpoint_path<tab>http_method (e.g., "/path/to/endpoint<tab>GET")
#             or just endpoint_path (defaults to GET for backward compatibility)
profile_endpoint() {
    local endpoint_input="$1"
    local endpoint
    local http_method
    
    # Parse endpoint and method (format: path<tab>method)
    if [[ "$endpoint_input" == *$'\t'* ]]; then
        IFS=$'\t' read -r endpoint http_method <<< "$endpoint_input"
        # Trim whitespace from method
        http_method=$(tr -d '[:space:]' <<< "$http_method")
    else
        # Backward compatibility: if no method specified, default to GET
        endpoint="$endpoint_input"
        http_method="GET"
    fi
    
    # Normalize method to uppercase
    http_method=$(tr '[:lower:]' '[:upper:]' <<< "$http_method")
    
    local endpoint_name
    endpoint_name=$(sanitize_endpoint "$endpoint")
    [ -z "$endpoint_name" ] && endpoint_name="endpoint"
    
    local substituted_path
    substituted_path=$(substitute_path_params "$endpoint")
    
    # Add api-version query parameter (required by ARM API)
    local test_url
    if [[ "$substituted_path" == *"?"* ]]; then
        test_url="${LOADTEST_BASE_URL}${substituted_path}&api-version=${TEST_API_VERSION}"
    else
        test_url="${LOADTEST_BASE_URL}${substituted_path}?api-version=${TEST_API_VERSION}"
    fi
    
    local seconds="${DURATION//s/}"
    
    echo "=========================================="
    echo "Profiling endpoint: $endpoint"
    echo "HTTP Method: $http_method"
    echo "Substituted path: $substituted_path"
    echo "API version: $TEST_API_VERSION"
    echo "URL: $test_url"
    echo "Duration: $DURATION"
    echo "Rate: $RATE req/s"
    echo "Profile prefix: $endpoint_name"
    echo "=========================================="
    echo ""
    
    # Check if pprof server is running
    if ! curl -s -o /dev/null -w "%{http_code}" "${PPROF_URL}/debug/pprof/" | grep -q "200"; then
        echo "Warning: pprof server is not running at ${PPROF_URL}"
        echo "Start it with: make runlocal-rp (with PPROF_ENABLED=true)"
        echo ""
    fi
    
    # Check if vegeta is available
    command -v vegeta >/dev/null 2>&1 || {
        echo "Error: vegeta not found. Install with: go install github.com/tsenart/vegeta@latest"
        exit 1
    }
    
    mkdir -p "$PPROF_OUTPUT_DIR"
    
    echo "Starting vegeta attack in background..."
    echo "Note: Most endpoints require authentication (mutual TLS or MISE) and valid resources."
    echo "      Expected errors (these are normal and indicate the server is processing requests):"
    echo "      - 400 Bad Request:"
    echo "        * Resource validation (e.g., OpenShift version not in enabled versions cache)"
    echo "        * InvalidSubscriptionState (subscription not registered in test environment)"
    echo "        * Missing request body for PUT/PATCH/POST endpoints"
    echo "      - 403 Forbidden: Authentication required (mutual TLS or MISE)"
    echo "      - 404 Not Found: Resource doesn't exist (expected for test data like test-cluster)"
    echo "      - 405 Method Not Allowed: Wrong HTTP method (should be fixed now)"
    echo "      Profiling will still capture server behavior under load regardless of response codes."
    echo ""
    
    # Build vegeta target with correct HTTP method
    # Format: METHOD URL (vegeta expects this format)
    # Pipe directly to vegeta - it reads from stdin
    {
        printf '%s %s\n' "$http_method" "$test_url"
    } | vegeta attack -duration="$DURATION" -rate="$RATE" -insecure > "${PPROF_OUTPUT_DIR}/${endpoint_name}-vegeta.bin" &
    local vegeta_pid=$!
    echo "Vegeta PID: $vegeta_pid"
    sleep 1
    echo ""
    
    echo "Collecting profiles during load test..."
    
    # CPU profile
    echo "  → CPU profile (${seconds} seconds)..."
    if curl -s "${PPROF_URL}/debug/pprof/profile?seconds=${seconds}" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-cpu.prof" 2>/dev/null; then
        echo "    ✓ CPU: ${PPROF_OUTPUT_DIR}/${endpoint_name}-cpu.prof"
    else
        echo "    ✗ Failed to collect CPU profile"
    fi
    
    # Heap profile
    echo "  → Heap profile..."
    if curl -s "${PPROF_URL}/debug/pprof/heap" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-heap.prof" 2>/dev/null; then
        echo "    ✓ Heap: ${PPROF_OUTPUT_DIR}/${endpoint_name}-heap.prof"
    else
        echo "    ✗ Failed to collect heap profile"
    fi
    
    # Allocs profile
    echo "  → Allocs profile..."
    if curl -s "${PPROF_URL}/debug/pprof/allocs" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-allocs.prof" 2>/dev/null; then
        echo "    ✓ Allocs: ${PPROF_OUTPUT_DIR}/${endpoint_name}-allocs.prof"
    else
        echo "    ✗ Failed to collect allocs profile"
    fi
    
    # Goroutine profile
    echo "  → Goroutine profile..."
    if curl -s "${PPROF_URL}/debug/pprof/goroutine" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-goroutine.prof" 2>/dev/null; then
        echo "    ✓ Goroutine: ${PPROF_OUTPUT_DIR}/${endpoint_name}-goroutine.prof"
    else
        echo "    ✗ Failed to collect goroutine profile"
    fi
    
    # Block profile
    echo "  → Block profile..."
    if curl -s "${PPROF_URL}/debug/pprof/block" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-block.prof" 2>/dev/null; then
        echo "    ✓ Block: ${PPROF_OUTPUT_DIR}/${endpoint_name}-block.prof"
    else
        echo "    ✗ Failed to collect block profile"
    fi
    
    # Mutex profile
    echo "  → Mutex profile..."
    if curl -s "${PPROF_URL}/debug/pprof/mutex" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-mutex.prof" 2>/dev/null; then
        echo "    ✓ Mutex: ${PPROF_OUTPUT_DIR}/${endpoint_name}-mutex.prof"
    else
        echo "    ✗ Failed to collect mutex profile"
    fi
    
    # Threadcreate profile
    echo "  → Threadcreate profile..."
    if curl -s "${PPROF_URL}/debug/pprof/threadcreate" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-threadcreate.prof" 2>/dev/null; then
        echo "    ✓ Threadcreate: ${PPROF_OUTPUT_DIR}/${endpoint_name}-threadcreate.prof"
    else
        echo "    ✗ Failed to collect threadcreate profile"
    fi
    
    # Execution trace
    echo "  → Execution trace (5s)..."
    if curl -s "${PPROF_URL}/debug/pprof/trace?seconds=5" -o "${PPROF_OUTPUT_DIR}/${endpoint_name}-trace.out" 2>/dev/null; then
        echo "    ✓ Trace: ${PPROF_OUTPUT_DIR}/${endpoint_name}-trace.out"
    else
        echo "    ✗ Failed to collect trace"
    fi
    
    echo ""
    echo "Waiting for vegeta to finish..."
    wait "$vegeta_pid" 2>/dev/null || true
    
    echo ""
    echo "Vegeta report:"
    vegeta report "${PPROF_OUTPUT_DIR}/${endpoint_name}-vegeta.bin" || true
    
    echo ""
    echo "=========================================="
    echo "Profile collection complete for: $endpoint"
    echo "=========================================="
    echo ""
}

# Main execution
main() {
    if [ -z "$ENDPOINT" ]; then
        echo "Error: ENDPOINT is required"
        echo "Usage: $0 ENDPOINT=/api/v1/clusters [DURATION=20s] [RATE=100]"
        echo "       $0 ENDPOINT=all [DURATION=10s] [RATE=50]"
        exit 1
    fi
    
    if [ "$ENDPOINT" = "all" ]; then
        echo "Extracting endpoints from swagger: $LATEST_SWAGGER"
        echo ""
        
        local endpoints
        endpoints=$(extract_endpoints_from_swagger "$LATEST_SWAGGER")
        
        if [ -z "$endpoints" ]; then
            echo "Error: No endpoints found in swagger file"
            exit 1
        fi
        
        local count
        count=$(echo "$endpoints" | wc -l | tr -d ' ')
        echo "Found $count endpoints to profile"
        echo ""
        echo "Note: After running 'make runlocal-rp', the RP frontend server is available."
        echo "      However, many endpoints require:"
        echo "      - Authentication (mutual TLS or MISE)"
        echo "      - Existing resources (subscriptions, clusters, etc.)"
        echo "      - Valid API versions"
        echo "      - Request bodies for PUT/PATCH/POST endpoints"
        echo ""
        echo "      Expected response codes (these are normal and indicate server processing):"
        echo "      - 400: Validation errors, missing resources, unregistered subscriptions"
        echo "      - 403: Authentication required"
        echo "      - 404: Resources don't exist (expected for test data)"
        echo "      - 405: Wrong HTTP method (should be fixed)"
        echo ""
        echo "      The profiling will still capture the server's behavior under load."
        echo ""
        read -p "Continue? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 0
        fi
        
        while IFS=$'\t' read -r endpoint method; do
            [ -z "$endpoint" ] && continue
            # Trim whitespace from method
            method=$(tr -d '[:space:]' <<< "${method:-GET}")
            echo ""
            echo ">>> Profiling: $endpoint ($method) <<<"
            echo ""
            profile_endpoint "${endpoint}$(printf '\t')${method}"
            echo ""
            echo "---"
            echo ""
            sleep 2  # Small delay between endpoints
        done <<< "$endpoints"
        
        echo ""
        echo "=========================================="
        echo "All endpoints profiled!"
        echo "=========================================="
        echo ""
        echo "View profiles in: $PPROF_OUTPUT_DIR"
        echo ""
        echo "Example commands:"
        echo "  go tool pprof -http=:8888 ${PPROF_OUTPUT_DIR}/<endpoint>-cpu.prof"
        echo "  go tool pprof -http=:8888 ${PPROF_OUTPUT_DIR}/<endpoint>-heap.prof"
        echo "  go tool trace ${PPROF_OUTPUT_DIR}/<endpoint>-trace.out"
    else
        profile_endpoint "$ENDPOINT"
        
        local endpoint_name
        endpoint_name=$(sanitize_endpoint "$ENDPOINT")
        echo "View profiles:"
        echo "  go tool pprof -http=:8888 ${PPROF_OUTPUT_DIR}/${endpoint_name}-cpu.prof"
        echo "  go tool pprof -http=:8888 ${PPROF_OUTPUT_DIR}/${endpoint_name}-heap.prof"
        echo "  go tool pprof -http=:8888 ${PPROF_OUTPUT_DIR}/${endpoint_name}-goroutine.prof"
        echo "  go tool trace ${PPROF_OUTPUT_DIR}/${endpoint_name}-trace.out"
        echo ""
        echo "Generate vegeta HTML report:"
        echo "  vegeta report -type=html ${PPROF_OUTPUT_DIR}/${endpoint_name}-vegeta.bin > ${PPROF_OUTPUT_DIR}/${endpoint_name}-vegeta.html"
    fi
}

main "$@"

