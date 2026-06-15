import { SystemHost } from "../core/types.js";
import { FileSystemCache } from "./file-system-cache.js";
import { ServerLog } from "./types.js";
export declare function resolveEntrypointFile(host: SystemHost, entrypoints: string[] | undefined | null, path: string, fileSystemCache: FileSystemCache | undefined, log: (log: ServerLog) => void): Promise<string | undefined>;
//# sourceMappingURL=entrypoint-resolver.d.ts.map