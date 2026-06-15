import type { Queue, Task } from '../../../types/index.js';
/**
 * Run the chained tasks and return the results
 */
export declare function toYaml<T extends Task[]>(queue: Queue<T>): Promise<string>;
//# sourceMappingURL=toYaml.d.ts.map