/**
 * PositionedMarker represents a marker in the code with its name and position.
 */
export interface PositionedMarker {
    /** Marker name */
    readonly name: string;
    /** Position of the marker */
    readonly pos: number;
}
/**
 * Extract TypeScript fourslash-style markers: /\*markerName*\/
 * @param code
 * @returns  an array of Marker objects with name, pos, and end
 */
export declare function extractMarkers(code: string): PositionedMarker[];
//# sourceMappingURL=fourslash.d.ts.map