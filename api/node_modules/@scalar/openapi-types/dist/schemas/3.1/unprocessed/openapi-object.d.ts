import { z } from 'zod';
import { ComponentsObjectSchema } from './components-object.js';
import { ExternalDocumentationObjectSchema } from './external-documentation-object.js';
import { InfoObjectSchema } from './info-object.js';
import { PathsObjectSchema } from './paths-object.js';
import { SecurityRequirementObjectSchema } from './security-requirement-object.js';
import { ServerObjectSchema } from './server-object.js';
import { TagObjectSchema } from './tag-object.js';
import { WebhooksObjectSchema } from './webhooks-object.js';
export type OpenApiObject = {
    openapi: string;
    info: z.infer<typeof InfoObjectSchema>;
    jsonSchemaDialect?: string;
    servers?: z.infer<typeof ServerObjectSchema>[];
    paths?: z.infer<typeof PathsObjectSchema>;
    webhooks?: z.infer<typeof WebhooksObjectSchema>;
    components?: z.infer<typeof ComponentsObjectSchema>;
    security?: z.infer<typeof SecurityRequirementObjectSchema>[];
    tags?: z.infer<typeof TagObjectSchema>[];
    externalDocs?: z.infer<typeof ExternalDocumentationObjectSchema>;
};
/**
 * OpenAPI Object
 *
 * This is the root object of the OpenAPI Description.
 *
 * @see https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#openapi-object
 */
export declare const OpenApiObjectSchema: z.ZodType<OpenApiObject>;
//# sourceMappingURL=openapi-object.d.ts.map