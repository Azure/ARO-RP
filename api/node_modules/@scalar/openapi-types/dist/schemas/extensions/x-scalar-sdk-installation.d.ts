import { z } from 'zod';
export declare const XScalarSdkInstallationSchema: z.ZodObject<{
    'x-scalar-sdk-installation': z.ZodCatch<z.ZodOptional<z.ZodArray<z.ZodObject<{
        lang: z.ZodString;
        source: z.ZodCatch<z.ZodOptional<z.ZodString>>;
        description: z.ZodCatch<z.ZodOptional<z.ZodString>>;
    }, z.core.$strip>>>>;
}, z.core.$strip>;
//# sourceMappingURL=x-scalar-sdk-installation.d.ts.map