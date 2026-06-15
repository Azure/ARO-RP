import { CompilerHost, Diagnostic } from "../../types.js";
export interface InfoCliArgs {
    emitter?: string;
}
/**
 * Print the resolved TypeSpec configuration, or emitter options if an emitter is specified.
 */
export declare function printInfoAction(host: CompilerHost, args: InfoCliArgs): Promise<readonly Diagnostic[]>;
//# sourceMappingURL=info.d.ts.map