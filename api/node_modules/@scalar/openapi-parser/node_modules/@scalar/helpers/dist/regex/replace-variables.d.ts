/**
 * This function takes a string and replaces both {single} and {{double}} curly brace variables with given values.
 * Use the replacePathVariables and replaceEnvVariables functions if you only need to replace one type of variable.
 */
export declare function replaceVariables(value: string, variablesOrCallback: Record<string, string | number> | ((match: string) => string)): string;
/** Replace {path} variables with their values */
export declare const replacePathVariables: (path: string, variables?: Record<string, string>) => string;
/** Replace {{env}} variables with their values */
export declare const replaceEnvVariables: (path: string, variables?: Record<string, string>) => string;
//# sourceMappingURL=replace-variables.d.ts.map