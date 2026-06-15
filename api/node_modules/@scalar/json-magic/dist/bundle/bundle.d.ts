import type { UnknownObject } from '../types.js';
/**
 * Checks if a string is a remote URL (starts with http:// or https://)
 * @param value - The URL string to check
 * @returns true if the string is a remote URL, false otherwise
 * @example
 * ```ts
 * isRemoteUrl('https://example.com/schema.json') // true
 * isRemoteUrl('http://api.example.com/schemas/user.json') // true
 * isRemoteUrl('#/components/schemas/User') // false
 * isRemoteUrl('./local-schema.json') // false
 * ```
 */
export declare function isRemoteUrl(value: string): boolean;
/**
 * Checks if a string represents a file path by ensuring it's not a remote URL,
 * YAML content, or JSON content.
 *
 * @param value - The string to check
 * @returns true if the string appears to be a file path, false otherwise
 * @example
 * ```ts
 * isFilePath('./schemas/user.json') // true
 * isFilePath('https://example.com/schema.json') // false
 * isFilePath('{"type": "object"}') // false
 * isFilePath('type: object') // false
 * ```
 */
export declare function isFilePath(value: string): boolean;
/**
 * Checks if a string is a local reference (starts with #)
 * @param value - The reference string to check
 * @returns true if the string is a local reference, false otherwise
 * @example
 * ```ts
 * isLocalRef('#/components/schemas/User') // true
 * isLocalRef('https://example.com/schema.json') // false
 * isLocalRef('./local-schema.json') // false
 * ```
 */
export declare function isLocalRef(value: string): boolean;
export type ResolveResult = {
    ok: true;
    data: unknown;
    raw: string;
} | {
    ok: false;
};
/**
 * Sets a value at a specified path in an object, creating intermediate objects/arrays as needed.
 * This function traverses the object structure and creates any missing intermediate objects
 * or arrays based on the path segments. If the next segment is a numeric string, it creates
 * an array instead of an object.
 *
 * ⚠️ Warning: Be careful with object keys that look like numbers (e.g. "123") as this function
 * will interpret them as array indices and create arrays instead of objects. If you need to
 * use numeric-looking keys, consider prefixing them with a non-numeric character.
 *
 * @param obj - The target object to set the value in
 * @param path - The JSON pointer path where the value should be set
 * @param value - The value to set at the specified path
 * @throws {Error} If attempting to set a value at the root path ('')
 *
 * @example
 * const obj = {}
 * setValueAtPath(obj, '/foo/bar/0', 'value')
 * // Result:
 * // {
 * //   foo: {
 * //     bar: ['value']
 * //   }
 * // }
 *
 * @example
 * const obj = { existing: { path: 'old' } }
 * setValueAtPath(obj, '/existing/path', 'new')
 * // Result:
 * // {
 * //   existing: {
 * //     path: 'new'
 * //   }
 * // }
 *
 * @example
 * // ⚠️ Warning: This will create an array instead of an object with key "123"
 * setValueAtPath(obj, '/foo/123/bar', 'value')
 * // Result:
 * // {
 * //   foo: [
 * //     undefined,
 * //     undefined,
 * //     undefined,
 * //     { bar: 'value' }
 * //   ]
 * // }
 */
export declare function setValueAtPath(obj: any, path: string, value: any): void;
/**
 * Prefixes an internal JSON reference with a given path prefix.
 * Takes a local reference (starting with #) and prepends the provided prefix segments.
 *
 * @param input - The internal reference string to prefix (must start with #)
 * @param prefix - Array of path segments to prepend to the reference
 * @returns The prefixed reference string
 * @throws Error if input is not a local reference
 * @example
 * prefixInternalRef('#/components/schemas/User', ['definitions'])
 * // Returns: '#/definitions/components/schemas/User'
 */
export declare function prefixInternalRef(input: string, prefix: string[]): string;
/**
 * Updates internal references in an object by adding a prefix to their paths.
 * Recursively traverses the input object and modifies any local $ref references
 * by prepending the given prefix to their paths. This is used when embedding external
 * documents to maintain correct reference paths relative to the main document.
 *
 * @param input - The object to update references in
 * @param prefix - Array of path segments to prepend to internal reference paths
 * @returns void
 * @example
 * ```ts
 * const input = {
 *   foo: {
 *     $ref: '#/components/schemas/User'
 *   }
 * }
 * prefixInternalRefRecursive(input, ['definitions'])
 * // Result:
 * // {
 * //   foo: {
 * //     $ref: '#/definitions/components/schemas/User'
 * //   }
 * // }
 * ```
 */
export declare function prefixInternalRefRecursive(input: unknown, prefix: string[]): void;
/**
 * Resolves and copies referenced values from a source document to a target document.
 * This function traverses the document and copies referenced values to the target document,
 * while tracking processed references to avoid duplicates. It only processes references
 * that belong to the same external document.
 *
 * @param targetDocument - The document to copy referenced values to
 * @param sourceDocument - The source document containing the references
 * @param referencePath - The JSON pointer path to the reference
 * @param externalRefsKey - The key used for external references (e.g. 'x-ext')
 * @param documentKey - The key identifying the external document
 * @param bundleLocalRefs - Also bundles the local refs
 * @param processedNodes - Set of already processed nodes to prevent duplicates
 * @example
 * ```ts
 * const source = {
 *   components: {
 *     schemas: {
 *       User: {
 *         $ref: '#/x-ext/users~1schema/definitions/Person'
 *       }
 *     }
 *   }
 * }
 *
 * const target = {}
 * resolveAndCopyReferences(
 *   target,
 *   source,
 *   '/components/schemas/User',
 *   'x-ext',
 *   'users/schema'
 * )
 * // Result: target will contain the User schema with resolved references
 * ```
 */
export declare const resolveAndCopyReferences: (targetDocument: unknown, sourceDocument: unknown, referencePath: string, externalRefsKey: string, documentKey: string, bundleLocalRefs?: boolean, processedNodes?: Set<unknown>) => void;
/**
 * A loader plugin for resolving external references during bundling.
 * Loader plugins are responsible for handling specific types of external references,
 * such as files, URLs, or custom protocols. Each loader plugin must provide:
 *
 * - A `validate` function to determine if the plugin can handle a given reference string.
 * - An `exec` function to asynchronously fetch and resolve the referenced data,
 *   returning a Promise that resolves to a `ResolveResult`.
 *
 * Loader plugins enable extensible support for different reference sources in the bundler.
 *
 * @property type - The plugin type, always 'loader' for loader plugins.
 * @property validate - Function to check if the plugin can handle a given reference value.
 * @property exec - Function to fetch and resolve the reference, returning the resolved data.
 */
export type LoaderPlugin = {
    type: 'loader';
    validate: (value: string) => boolean;
    exec: (value: string) => Promise<ResolveResult>;
};
/**
 * Context information for a node during traversal or processing.
 *
 * Note: The `path` parameter represents the path to the current node being processed.
 * If you are performing a partial bundle (i.e., providing a custom root), this path will be relative
 * to the root you provide, not the absolute root of the original document. You may need to prefix
 * it with your own base path if you want to construct a full path from the absolute document root.
 *
 * - `path`: The JSON pointer path (as an array of strings) from the document root to the current node.
 * - `resolutionCache`: A cache for storing promises of resolved references.
 */
type NodeProcessContext = {
    path: readonly string[];
    resolutionCache: Map<string, Promise<Readonly<ResolveResult>>>;
    parentNode: UnknownObject | null;
    rootNode: UnknownObject;
    loaders: LoaderPlugin[];
};
/**
 * A plugin type for lifecycle hooks, allowing custom logic to be injected into the bundler's process.
 * This type extends the Config['hooks'] interface and is identified by type: 'lifecycle'.
 */
export type LifecyclePlugin = {
    type: 'lifecycle';
} & Config['hooks'];
/**
 * Represents a plugin used by the bundler for extensibility.
 *
 * Plugins can be either:
 * - Loader plugins: Responsible for resolving and loading external references (e.g., from files, URLs, or custom sources).
 * - Lifecycle plugins: Provide lifecycle hooks to customize or extend the bundling process.
 *
 * Loader plugins must implement:
 *   - `validate`: Checks if the plugin can handle a given reference value.
 *   - `exec`: Asynchronously resolves and returns the referenced data.
 *
 * Lifecycle plugins extend the bundler's lifecycle hooks for custom logic.
 */
export type Plugin = LoaderPlugin | LifecyclePlugin;
/**
 * Configuration options for the bundler.
 * Controls how external references are resolved and processed during bundling.
 */
type Config = {
    /**
     * Array of plugins that handle resolving references from different sources.
     * Each plugin is responsible for fetching and processing data from specific sources
     * like URLs or the filesystem.
     */
    plugins: Plugin[];
    /**
     * Optional root object that serves as the base document when bundling a subpart.
     * This allows resolving references relative to the root document's location,
     * ensuring proper path resolution for nested references.
     */
    root?: UnknownObject;
    /**
     * Optional maximum depth for reference resolution.
     * Limits how deeply the bundler will follow and resolve nested $ref pointers.
     * Useful for preventing infinite recursion or excessive resource usage.
     */
    depth?: number;
    /**
     * Optional origin path for the bundler.
     * Used to resolve relative paths in references, especially when the input is a string URL or file path.
     * If not provided, the bundler will use the input value as the origin.
     */
    origin?: string;
    /**
     * Optional cache to store promises of resolved references.
     * Helps avoid duplicate fetches/reads of the same resource by storing
     * the resolution promises for reuse.
     */
    cache?: Map<string, Promise<ResolveResult>>;
    /**
     * Cache of visited nodes during partial bundling.
     * Used to prevent re-bundling the same tree multiple times when doing partial bundling,
     * improving performance by avoiding redundant processing of already bundled sections.
     */
    visitedNodes?: Set<unknown>;
    /**
     * Enable tree shaking to optimize the bundle size.
     * When enabled, only the parts of external documents that are actually referenced
     * will be included in the final bundle.
     */
    treeShake: boolean;
    /**
     * Optional flag to generate a URL map.
     * When enabled, tracks the original source URLs of bundled references
     * in an section for reference mapping defined by externalDocumentsMappingsKey.
     */
    urlMap?: boolean;
    /**
     * Custom OpenAPI extension key used to store external references.
     * This key will contain all bundled external documents.
     * The key is used to maintain a clean separation between the main
     * OpenAPI document and its bundled external references.
     * @default 'x-ext'
     */
    externalDocumentsKey?: string;
    /**
     * Custom OpenAPI extension key used to maintain a mapping between
     * hashed keys and their original URLs.
     * This mapping is essential for tracking the source of bundled references
     * @default 'x-ext-urls'
     */
    externalDocumentsMappingsKey?: string;
    /**
     * Optional function to compress input URLs or file paths before bundling.
     * Returns either a Promise resolving to the compressed string or the compressed string directly.
     */
    compress?: (value: string) => Promise<string> | string;
    /**
     * Optional hooks to monitor the bundler's lifecycle.
     * Allows tracking the progress and status of reference resolution.
     */
    hooks?: Partial<{
        /**
         * Optional hook called when the bundler starts resolving a $ref.
         * Useful for tracking or logging the beginning of a reference resolution.
         */
        onResolveStart: (node: UnknownObject & Record<'$ref', unknown>) => void;
        /**
         * Optional hook called when the bundler fails to resolve a $ref.
         * Can be used for error handling, logging, or custom error reporting.
         */
        onResolveError: (node: UnknownObject & Record<'$ref', unknown>) => void;
        /**
         * Optional hook called when the bundler successfully resolves a $ref.
         * Useful for tracking successful resolutions or custom post-processing.
         */
        onResolveSuccess: (node: UnknownObject & Record<'$ref', unknown>) => void;
        /**
         * Optional hook invoked before processing a node.
         * Can be used for preprocessing, mutation, or custom logic before the node is handled by the bundler.
         */
        onBeforeNodeProcess: (node: UnknownObject, context: NodeProcessContext) => void | Promise<void>;
        /**
         * Optional hook invoked after processing a node.
         * Useful for postprocessing, cleanup, or custom logic after the node has been handled by the bundler.
         */
        onAfterNodeProcess: (node: UnknownObject, context: NodeProcessContext) => void | Promise<void>;
    }>;
};
/**
 * Extension keys used for bundling external references in OpenAPI documents.
 * These custom extensions help maintain the structure and traceability of bundled documents.
 */
export declare const extensions: {
    /**
     * Custom OpenAPI extension key used to store external references.
     * This key will contain all bundled external documents.
     * The x-ext key is used to maintain a clean separation between the main
     * OpenAPI document and its bundled external references.
     */
    readonly externalDocuments: "x-ext";
    /**
     * Custom OpenAPI extension key used to maintain a mapping between
     * hashed keys and their original URLs in x-ext.
     * This mapping is essential for tracking the source of bundled references
     */
    readonly externalDocumentsMappings: "x-ext-urls";
};
/**
 * Bundles an OpenAPI specification by resolving all external references.
 * This function traverses the input object recursively and embeds external $ref
 * references into an x-ext section. External references can be URLs or local files.
 * The original $refs are updated to point to their embedded content in the x-ext section.
 * If the input is an object, it will be modified in place by adding an x-ext
 * property to store resolved external references.
 *
 * @param input - The OpenAPI specification to bundle. Can be either an object or string.
 *                If a string is provided, it will be resolved using the provided plugins.
 *                If no plugin can process the input, the onReferenceError hook will be invoked
 *                and an error will be emitted to the console.
 * @param config - Configuration object containing plugins and options for bundling OpenAPI specifications
 * @returns A promise that resolves to the bundled specification with all references embedded
 * @example
 * // Example with object input
 * const spec = {
 *   paths: {
 *     '/users': {
 *       $ref: 'https://example.com/schemas/users.yaml'
 *     }
 *   }
 * }
 *
 * const bundled = await bundle(spec, {
 *   plugins: [fetchUrls()],
 *   treeShake: true,
 *   urlMap: true,
 *   hooks: {
 *     onResolveStart: (ref) => console.log('Resolving:', ref.$ref),
 *     onResolveSuccess: (ref) => console.log('Resolved:', ref.$ref),
 *     onResolveError: (ref) => console.log('Failed to resolve:', ref.$ref)
 *   }
 * })
 * // Result:
 * // {
 * //   paths: {
 * //     '/users': {
 * //       $ref: '#/x-ext/abc123'
 * //     }
 * //   },
 * //   'x-ext': {
 * //     'abc123': {
 * //       // Resolved content from users.yaml
 * //     }
 * //   },
 * //   'x-ext-urls': {
 * //     'https://example.com/schemas/users.yaml': 'abc123'
 * //   }
 * // }
 *
 * // Example with URL input
 * const bundledFromUrl = await bundle('https://example.com/openapi.yaml', {
 *   plugins: [fetchUrls()],
 *   treeShake: true,
 *   urlMap: true,
 *   hooks: {
 *     onResolveStart: (ref) => console.log('Resolving:', ref.$ref),
 *     onResolveSuccess: (ref) => console.log('Resolved:', ref.$ref),
 *     onResolveError: (ref) => console.log('Failed to resolve:', ref.$ref)
 *   }
 * })
 * // The function will first fetch the OpenAPI spec from the URL,
 * // then bundle all its external references into the x-ext section
 */
export declare function bundle(input: UnknownObject | string, config: Config): Promise<object>;
export {};
//# sourceMappingURL=bundle.d.ts.map