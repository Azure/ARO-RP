/**
 * Extracts secret fields from the config or the old schemes
 * Maps original field names to their x-scalar-secret extension equivalents.
 */
export declare const extractConfigSecrets: (input: Record<string, unknown>) => Record<string, string>;
/** Removes all secret fields from the input object */
export declare const removeSecretFields: (input: Record<string, unknown>) => Record<string, unknown>;
//# sourceMappingURL=extract-config-secrets.d.ts.map