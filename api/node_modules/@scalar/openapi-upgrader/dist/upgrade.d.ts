import type { OpenAPIV3, OpenAPIV3_1, OpenAPIV3_2 } from '@scalar/openapi-types';
import type { UnknownObject } from '@scalar/types/utils';
/**
 * Upgrade OpenAPI documents from Swagger 2.0 or OpenAPI 3.0 to the specified target version
 */
export declare function upgrade(value: UnknownObject, targetVersion: '3.0'): OpenAPIV3.Document;
export declare function upgrade(value: UnknownObject, targetVersion: '3.1'): OpenAPIV3_1.Document;
export declare function upgrade(value: UnknownObject, targetVersion: '3.2'): OpenAPIV3_2.Document;
//# sourceMappingURL=upgrade.d.ts.map