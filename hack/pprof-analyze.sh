#!/bin/bash
# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.
#
# Analyze pprof profiles and generate improvement suggestions.
# Usage:
#   ./hack/pprof-analyze.sh <profile-file>
#   ./hack/pprof-analyze.sh pprof-data/providers-microsoft-redhatopenshift-operations-cpu.prof

set -euo pipefail

PROFILE="${1:-}"
PPROF_OUTPUT_DIR="${PPROF_OUTPUT_DIR:-./pprof-data}"

if [ -z "$PROFILE" ]; then
    echo "Usage: $0 <profile-file>"
    echo ""
    echo "Available profiles:"
    ls -1 "$PPROF_OUTPUT_DIR"/*.prof 2>/dev/null | sed 's|.*/||' | sed 's|^|  |' || echo "  No profiles found in $PPROF_OUTPUT_DIR"
    exit 1
fi

if [ ! -f "$PROFILE" ]; then
    echo "Error: Profile file not found: $PROFILE"
    exit 1
fi

ENDPOINT_NAME=$(basename "$PROFILE" .prof)
REPORT_FILE="${PPROF_OUTPUT_DIR}/${ENDPOINT_NAME}-analysis.md"

echo "Analyzing profile: $PROFILE"
echo "Generating report: $REPORT_FILE"
echo ""

{
    echo "# Performance Analysis: $ENDPOINT_NAME"
    echo ""
    echo "Generated: $(date)"
    echo "Profile: $PROFILE"
    echo ""
    echo "---"
    echo ""
    
    # Determine profile type
    if [[ "$PROFILE" == *"-cpu.prof" ]]; then
        echo "## Profile Type: CPU"
        echo ""
        echo "### Top Functions by CPU Time (Cumulative)"
        echo '```'
        go tool pprof -top -cum "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Top Functions by CPU Time (Flat)"
        echo '```'
        go tool pprof -top "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Key Insights"
        echo ""
        echo "**Hot Paths**: Functions with high cumulative time are in the critical path"
        echo "**Bottlenecks**: Functions with high flat time are doing actual work"
        echo ""
        
    elif [[ "$PROFILE" == *"-heap.prof" ]]; then
        echo "## Profile Type: Heap (In-Use Memory)"
        echo ""
        echo "### Top Allocations by Size (In-Use Space)"
        echo '```'
        go tool pprof -top -cum -inuse_space "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Top Allocations by Count (In-Use Objects)"
        echo '```'
        go tool pprof -top -cum -inuse_objects "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Key Insights"
        echo ""
        echo "**Memory Usage**: Shows currently allocated memory"
        echo "**Potential Leaks**: Look for unexpected allocations that persist"
        echo ""
        
    elif [[ "$PROFILE" == *"-allocs.prof" ]]; then
        echo "## Profile Type: Allocations (Total Since Start)"
        echo ""
        echo "### Top Allocations by Total Space"
        echo '```'
        go tool pprof -top -cum -alloc_space "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Top Allocations by Total Count"
        echo '```'
        go tool pprof -top -cum -alloc_objects "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Key Insights"
        echo ""
        echo "**Allocation Hotspots**: Functions that allocate frequently"
        echo "**GC Pressure**: High allocation rates can cause GC pauses"
        echo ""
        
    elif [[ "$PROFILE" == *"-goroutine.prof" ]]; then
        echo "## Profile Type: Goroutines"
        echo ""
        echo "### Goroutine Count"
        echo '```'
        go tool pprof -top "$PROFILE" 2>&1 | head -5
        echo '```'
        echo ""
        echo "### Top Goroutine Stack Traces"
        echo '```'
        go tool pprof -top "$PROFILE" 2>&1 | head -50
        echo '```'
        echo ""
        echo "### Key Insights"
        echo ""
        echo "**Goroutine Leaks**: Unusually high goroutine counts"
        echo "**Blocking Operations**: Goroutines stuck in waiting states"
        echo ""
        
    elif [[ "$PROFILE" == *"-block.prof" ]]; then
        echo "## Profile Type: Block (Synchronization Blocking)"
        echo ""
        echo "### Top Blocking Operations"
        echo '```'
        go tool pprof -top -cum "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Key Insights"
        echo ""
        echo "**Lock Contention**: Time spent waiting on locks"
        echo "**Channel Blocking**: Time spent waiting on channel operations"
        echo ""
        
    elif [[ "$PROFILE" == *"-mutex.prof" ]]; then
        echo "## Profile Type: Mutex (Lock Contention)"
        echo ""
        echo "### Top Mutex Contention"
        echo '```'
        go tool pprof -top -cum "$PROFILE" 2>&1 | head -30
        echo '```'
        echo ""
        echo "### Key Insights"
        echo ""
        echo "**Lock Contention**: Mutexes with high contention"
        echo "**Performance Impact**: Time other goroutines waited for locks"
        echo ""
    else
        echo "## Profile Type: Unknown"
        echo ""
        echo "### Top Functions"
        echo '```'
        go tool pprof -top -cum "$PROFILE" 2>&1 | head -30
        echo '```'
    fi
    
    echo ""
    echo "---"
    echo ""
    echo "## Improvement Suggestions"
    echo ""
    echo "### 1. Review Hot Paths"
    echo "- Identify functions with highest CPU/memory usage"
    echo "- Consider caching, optimization, or algorithm improvements"
    echo ""
    echo "### 2. Check for Allocation Patterns"
    echo "- Look for frequent allocations in hot paths"
    echo "- Consider object pooling or pre-allocation"
    echo ""
    echo "### 3. Analyze Concurrency"
    echo "- Review goroutine counts and states"
    echo "- Check for potential leaks or excessive concurrency"
    echo ""
    echo "### 4. Investigate Blocking"
    echo "- Review lock contention and blocking operations"
    echo "- Consider lock-free alternatives or reducing lock scope"
    echo ""
    echo "---"
    echo ""
    echo "## Next Steps"
    echo ""
    echo "1. Open interactive view:"
    echo "   \`\`\`bash"
    echo "   go tool pprof -http=:8888 $PROFILE"
    echo "   \`\`\`"
    echo ""
    echo "2. Compare with other profiles:"
    echo "   \`\`\`bash"
    echo "   go tool pprof -base=<other-profile> $PROFILE"
    echo "   \`\`\`"
    echo ""
    echo "3. Generate visualizations:"
    echo "   \`\`\`bash"
    echo "   go tool pprof -svg $PROFILE > ${ENDPOINT_NAME}-graph.svg"
    echo "   \`\`\`"
    
} > "$REPORT_FILE"

echo "✓ Analysis complete!"
echo ""
echo "View the report:"
echo "  cat $REPORT_FILE"
echo ""
echo "Or open interactive view:"
echo "  go tool pprof -http=:8888 $PROFILE"

