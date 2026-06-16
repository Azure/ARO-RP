import { z } from 'zod';
/** Options for the x-usePkce extension */
export declare const XUsePkceValues: readonly ["SHA-256", "plain", "no"];
export declare const XusePkceSchema: z.ZodObject<{
    'x-usePkce': z.ZodDefault<z.ZodOptional<z.ZodEnum<{
        "SHA-256": "SHA-256";
        plain: "plain";
        no: "no";
    }>>>;
}, z.core.$strip>;
//# sourceMappingURL=x-use-pkce.d.ts.map