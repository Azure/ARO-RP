import { z } from 'zod';
/**
 * x-enumDescriptions
 *
 * Maps enum values to their descriptions. Each key should correspond to
 * an enum value, and the value is the description for that enum value.
 *
 * Example:
 * x-enumDescriptions:
 *   missing_features: "Missing features"
 *   too_expensive: "Too expensive"
 *   unused: "Unused"
 *   other: "Other"
 */
export declare const XEnumDescriptionsSchema: z.ZodObject<{
    'x-enumDescriptions': z.ZodCatch<z.ZodRecord<z.ZodString, z.ZodString>>;
}, z.core.$strip>;
//# sourceMappingURL=x-enum-descriptions.d.ts.map