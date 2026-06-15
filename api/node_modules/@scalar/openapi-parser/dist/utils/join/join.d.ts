import type { OpenAPIV3_1 } from '@scalar/openapi-types';
import type { UnknownObject } from '../../types/index.js';
type Conflicts = {
    type: 'path';
    path: string;
    method: string;
} | {
    type: 'webhook';
    path: string;
    method: string;
} | {
    type: 'component';
    componentType: string;
    name: string;
};
type JoinResult = {
    ok: true;
    document: OpenAPIV3_1.Document;
} | {
    ok: false;
    conflicts: Conflicts[];
};
/**
 * Joins multiple OpenAPI documents into a single document.
 *
 * - Merges the "info" object, paths, webhooks, tags, and servers from all input documents.
 * - If there are conflicting paths or webhooks (same path and method), returns a list of conflicts.
 * - Only the first occurrence of a tag (by name) or server (by url) is included.
 * - The merge is performed in reverse order, so the first document in the input array has the highest precedence.
 *
 * @param inputs - Array of OpenAPI documents (UnknownObject) to join
 * @returns {JoinResult} - { ok: true, document } if successful, or { ok: false, conflicts } if there are conflicts
 *
 * @example
 * const doc1 = {
 *   info: { title: "API 1", version: "1.0.0" },
 *   paths: { "/foo": { get: { summary: "Get Foo" } } },
 *   tags: [{ name: "foo" }],
 *   servers: [{ url: "https://api1.example.com" }]
 * }
 * const doc2 = {
 *   info: { description: "Second API" },
 *   paths: { "/bar": { get: { summary: "Get Bar" } } },
 *   tags: [{ name: "bar" }],
 *   servers: [{ url: "https://api2.example.com" }]
 * }
 * const result = join([doc1, doc2])
 * // result.ok === true
 * // result.document.info.title === "API 1"
 * // result.document.info.description === "Second API"
 * // result.document.paths has both "/foo" and "/bar"
 * // result.document.tags contains both "foo" and "bar"
 * // result.document.servers contains both server URLs
 */
export declare const join: (inputs: UnknownObject[], config?: {
    prefixComponents: string[];
}) => Promise<JoinResult>;
export {};
//# sourceMappingURL=join.d.ts.map