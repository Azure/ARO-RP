import type { Queue, Task } from '../../../types/index.js';
import { details as detailsUtility } from '../../../utils/details.js';
/**
 * Run the chained tasks and return just some basic information about the OpenAPI document
 */
export declare function details<T extends Task[]>(queue: Queue<T>): Promise<ReturnType<typeof detailsUtility>>;
//# sourceMappingURL=details.d.ts.map