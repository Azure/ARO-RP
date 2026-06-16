import type { FilterResult, Queue, Task } from '../../../types/index.js';
import type { DereferenceOptions } from '../../../utils/dereference.js';
import type { FilterCallback } from '../../../utils/filter.js';
declare global {
    interface Commands {
        filter: {
            task: {
                name: 'filter';
                options?: FilterCallback;
            };
            result: FilterResult;
        };
    }
}
/**
 * Filter the given OpenAPI document
 */
export declare function filterCommand<T extends Task[]>(previousQueue: Queue<T>, options?: FilterCallback): {
    dereference: (dereferenceOptions?: DereferenceOptions) => {
        details: () => Promise<import("../../../types/index.js").DetailsResult>;
        files: () => Promise<import("../../../types/index.js").Filesystem>;
        get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
            name: "filter";
            options?: FilterCallback;
        }, {
            name: "dereference";
            options?: DereferenceOptions;
        }]>>;
        toJson: () => Promise<string>;
        toYaml: () => Promise<string>;
    };
    details: () => Promise<import("../../../types/index.js").DetailsResult>;
    files: () => Promise<import("../../../types/index.js").Filesystem>;
    get: () => Promise<import("../../../types/index.js").CommandChain<[...T, {
        name: "filter";
        options?: FilterCallback;
    }]>>;
    toJson: () => Promise<string>;
    toYaml: () => Promise<string>;
};
//# sourceMappingURL=filterCommand.d.ts.map