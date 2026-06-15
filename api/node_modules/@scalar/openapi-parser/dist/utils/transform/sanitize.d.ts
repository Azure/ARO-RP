import type { OpenAPI } from '@scalar/openapi-types';
import type { AnyObject } from '../../types/index.js';
export { DEFAULT_TITLE } from './utils/addInfoObject.js';
export { DEFAULT_OPENAPI_VERSION } from './utils/addLatestOpenApiVersion.js';
/**
 * Make an OpenAPI document a valid and clean OpenAPI document
 *
 * @deprecated We're about to drop this from the package.
 */
export declare function sanitize(definition: AnyObject): OpenAPI.Document;
//# sourceMappingURL=sanitize.d.ts.map