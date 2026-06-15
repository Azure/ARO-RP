/**
 * Collection of regular expressions used throughout the application.
 * These patterns handle URL parsing, variable detection, and reference path extraction.
 */
export declare const REGEX: {
    /** Checks for a valid scheme */
    readonly PROTOCOL: RegExp;
    /** Finds multiple slashes after the scheme to replace with a single slash */
    readonly MULTIPLE_SLASHES: RegExp;
    /** Finds all variables wrapped in {{double}} */
    readonly VARIABLES: RegExp;
    /** Finds all variables wrapped in {single} */
    readonly PATH: RegExp;
    /** Finds the name of the schema from the ref path */
    readonly REF_NAME: RegExp;
    /** Finds template variables in multiple formats: {{var}}, {var}, or :var */
    readonly TEMPLATE_VARIABLE: RegExp;
};
//# sourceMappingURL=regex-helpers.d.ts.map