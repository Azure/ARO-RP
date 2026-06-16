import type { UnknownObject } from '@scalar/types/utils';
/**
 * Recursively traverses the content and applies the transform function to each node.
 */
export declare function traverse(content: UnknownObject, transform: (content: UnknownObject, path?: string[]) => UnknownObject, path?: string[]): UnknownObject;
//# sourceMappingURL=traverse.d.ts.map