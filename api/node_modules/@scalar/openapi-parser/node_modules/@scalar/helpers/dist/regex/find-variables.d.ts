/**
 * Find all strings wrapped in {} or {{}} in value.
 *
 * @param value - The string to find variables in
 * @param includePath - Whether to include path variables {single}
 * @param includeEnv - Whether to include environment variables {{double}}
 */
export declare const findVariables: (value: string, { includePath, includeEnv }?: {
    includePath?: boolean;
    includeEnv?: boolean;
}) => string[];
//# sourceMappingURL=find-variables.d.ts.map