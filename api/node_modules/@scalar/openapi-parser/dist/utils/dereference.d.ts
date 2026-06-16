import type { AnyApiDefinitionFormat, DereferenceResult, Filesystem } from '../types/index.js';
import { type ResolveReferencesOptions } from './resolve-references.js';
export type DereferenceOptions = ResolveReferencesOptions;
/**
 * Dereferences an API definition or filesystem by resolving all references within the specification.
 *
 * @param value - The API definition or filesystem to dereference.\
 *                Can be any supported API definition format or a filesystem object.
 *
 * @param options - Optional options for the dereferencing process.
 */
export declare function dereference(value: AnyApiDefinitionFormat | Filesystem, options?: DereferenceOptions): DereferenceResult;
//# sourceMappingURL=dereference.d.ts.map