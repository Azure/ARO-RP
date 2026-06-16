import type { OpenAPIV3_1 } from '@scalar/openapi-types';
import type { UnknownObject } from '@scalar/types/utils';
/** Determine if the current path is within a schema - optimized version */
export declare function isSchemaPath(path: string[] | undefined): boolean;
/**
 * Upgrade from OpenAPI 3.0.x to 3.1.1
 *
 * https://www.openapis.org/blog/2021/02/16/migrating-from-openapi-3-0-to-3-1-0
 */
export declare function upgradeFromThreeToThreeOne(originalContent: UnknownObject): UnknownObject | (Omit<Omit<import("@scalar/openapi-types").OpenAPIV3.Document<{}>, "components" | "paths">, keyof {
    [customExtension: `x-${string}`]: any;
    [key: string]: any;
}> & {
    openapi?: "3.1.0" | "3.1.1" | "3.1.2";
    swagger?: never;
    info?: OpenAPIV3_1.InfoObject;
    jsonSchemaDialect?: string;
    servers?: OpenAPIV3_1.ServerObject[];
} & Pick<OpenAPIV3_1.PathsWebhooksComponents<{}>, "paths"> & Omit<Partial<OpenAPIV3_1.PathsWebhooksComponents<{}>>, "paths"> & {
    [customExtension: `x-${string}`]: any;
    [key: string]: any;
});
//# sourceMappingURL=upgrade-from-three-to-three-one.d.ts.map