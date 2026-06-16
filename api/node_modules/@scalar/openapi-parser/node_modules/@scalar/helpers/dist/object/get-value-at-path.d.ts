/**
 * Retrieves a nested value from the source document using a path array
 *
 * @example
 * ```ts
 * getValueByPath(document, ['components', 'schemas', 'User'])
 *
 * { id: '123', name: 'John Doe' }
 * ```
 */
export declare function getValueAtPath<R = unknown>(obj: any, pointer: string[]): R;
//# sourceMappingURL=get-value-at-path.d.ts.map