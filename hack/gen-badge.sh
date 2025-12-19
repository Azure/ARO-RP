#!/bin/bash
# hack/gen-badge.sh
# Usage: ./hack/gen-badge.sh <coverage_percentage> > coverage.svg

COVERAGE=${1:-0}
COLOR="red"

# Determine color based on coverage
if (($(echo "$COVERAGE < 50" | bc -l))); then
	COLOR="#e05d44" # Red
elif (($(echo "$COVERAGE < 80" | bc -l))); then
	COLOR="#dfb317" # Yellow
else
	COLOR="#4c1" # Green
fi

# Generate SVG
cat <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="100" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h60v20H0z"/>
    <path fill="${COLOR}" d="M60 0h40v20H60z"/>
    <path fill="url(#b)" d="M0 0h100v20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="30" y="15" fill="#010101" fill-opacity=".3">coverage</text>
    <text x="30" y="14">coverage</text>
    <text x="80" y="15" fill="#010101" fill-opacity=".3">${COVERAGE}%</text>
    <text x="80" y="14">${COVERAGE}%</text>
  </g>
</svg>
EOF
