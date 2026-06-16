import { z } from 'zod';
export declare const XScalarStabilityValues: {
    readonly Deprecated: "deprecated";
    readonly Experimental: "experimental";
    readonly Stable: "stable";
};
export type XScalarStabilityValues = (typeof XScalarStabilityValues)[keyof typeof XScalarStabilityValues];
/**
 * An OpenAPI extension to indicate the stability of the operation
 *
 * @example
 * ```yaml
 * x-scalar-stability: deprecated
 * ```
 */
export declare const XScalarStabilitySchema: z.ZodObject<{
    'x-scalar-stability': z.ZodCatch<z.ZodOptional<z.ZodEnum<{
        [x: string]: string;
    }>>>;
}, z.core.$strip>;
//# sourceMappingURL=x-scalar-stability.d.ts.map