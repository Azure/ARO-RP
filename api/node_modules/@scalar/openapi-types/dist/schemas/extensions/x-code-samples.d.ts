import { z } from 'zod';
export declare const XCodeSampleSchema: z.ZodObject<{
    lang: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    label: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    source: z.ZodString;
}, z.core.$strip>;
export declare const XCodeSamplesSchema: z.ZodObject<{
    'x-codeSamples': z.ZodCatch<z.ZodOptional<z.ZodArray<z.ZodObject<{
        lang: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        label: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        source: z.ZodString;
    }, z.core.$strip>>>>;
    'x-code-samples': z.ZodCatch<z.ZodOptional<z.ZodArray<z.ZodObject<{
        lang: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        label: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        source: z.ZodString;
    }, z.core.$strip>>>>;
    'x-custom-examples': z.ZodCatch<z.ZodOptional<z.ZodArray<z.ZodObject<{
        lang: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        label: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        source: z.ZodString;
    }, z.core.$strip>>>>;
}, z.core.$strip>;
export type XCodeSample = z.infer<typeof XCodeSampleSchema>;
//# sourceMappingURL=x-code-samples.d.ts.map