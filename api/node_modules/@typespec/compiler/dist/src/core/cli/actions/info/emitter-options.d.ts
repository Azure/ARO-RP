import type { PackageJson } from "../../../../types/package-json.js";
import { CompilerHost, Diagnostic } from "../../../types.js";
interface EmitterOptionVariant {
    type: string;
    allowedValues?: string[];
    default?: string;
    description?: string;
    nestedOptions?: EmitterOptionInfo[];
}
interface EmitterOptionInfo {
    name: string;
    type: string;
    allowedValues?: string[];
    default?: string;
    description?: string;
    nestedOptions?: EmitterOptionInfo[];
    /** When present, this option is a union of multiple variants */
    variants?: EmitterOptionVariant[];
}
/**
 * Extract option information from a JSON Schema properties object.
 * This is a pure function that can be tested independently.
 */
export declare function extractEmitterOptionsInfo(schema: any): EmitterOptionInfo[];
/**
 * Format library metadata (name, version, description, homepage) as colorized key-value lines
 * under a section title.
 */
export declare function formatLibraryInfo(manifest: PackageJson | undefined): string[];
/**
 * Format emitter options as a colorized string for terminal display.
 * Returns lines of formatted output.
 */
export declare function formatEmitterOptions(schema: any): string[];
/**
 * Resolve a library and print its info and emitter options.
 */
export declare function printEmitterOptionsAction(host: CompilerHost, emitterName: string): Promise<readonly Diagnostic[]>;
export {};
//# sourceMappingURL=emitter-options.d.ts.map