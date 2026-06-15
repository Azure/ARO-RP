import type { DereferenceResult, Queue, Task } from '../../../types/index.js';
import type { DereferenceOptions } from '../../../utils/dereference.js';
declare global {
    interface Commands {
        dereference: {
            task: {
                name: 'dereference';
                options?: DereferenceOptions;
            };
            result: DereferenceResult;
        };
    }
}
/**
 * Dereference the given OpenAPI document
 */
export declare function dereferenceCommand<T extends Task[]>(previousQueue: Queue<T>, options?: DereferenceOptions): {
    details: () => Promise<import("../../../types/index.js").DetailsResult>;
    files: () => Promise<import("../../../types/index.js").Filesystem>;
    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
        name: "dereference";
        options?: DereferenceOptions;
    }]>>;
    toJson: () => Promise<string>;
    toYaml: () => Promise<string>;
};
//# sourceMappingURL=dereferenceCommand.d.ts.map