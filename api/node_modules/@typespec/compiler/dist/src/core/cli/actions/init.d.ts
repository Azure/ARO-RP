import { Diagnostic } from "../../types.js";
import { CliCompilerHost } from "../types.js";
export interface InitArgs {
    templatesUrl?: string;
    template?: string;
    "no-prompt"?: boolean;
    args?: string[];
    "project-name"?: string;
    emitters?: string[];
    outputDir?: string;
}
export declare function initAction(host: CliCompilerHost, args: InitArgs): Promise<readonly Diagnostic[]>;
//# sourceMappingURL=init.d.ts.map