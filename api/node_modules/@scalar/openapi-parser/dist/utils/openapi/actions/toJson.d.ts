import type { Queue, Task } from '../../../types/index.js';
/**
 * Run the chained tasks and return the results
 */
export declare function toJson<T extends Task[]>(queue: Queue<T>): Promise<string | undefined>;
//# sourceMappingURL=toJson.d.ts.map