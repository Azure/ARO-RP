/**
 * Extract TypeScript fourslash-style markers: /\*markerName*\/
 * @param code
 * @returns  an array of Marker objects with name, pos, and end
 */
export function extractMarkers(code) {
    const markerRegex = /\/\*([a-zA-Z0-9_]+)\*\//g;
    const markers = [];
    let match;
    while ((match = markerRegex.exec(code)) !== null) {
        const markerName = match[1];
        const pos = markerRegex.lastIndex;
        markers.push({ name: markerName, pos });
    }
    return markers;
}
//# sourceMappingURL=fourslash.js.map