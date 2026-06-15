import type { AnyApiDefinitionFormat, AnyObject, FilterResult } from '../types/index.js';
export type FilterCallback = (schema: AnyObject) => boolean;
/**
 * Filter the specification based on the callback
 */
export declare function filter(specification: AnyApiDefinitionFormat, callback: FilterCallback): FilterResult;
//# sourceMappingURL=filter.d.ts.map