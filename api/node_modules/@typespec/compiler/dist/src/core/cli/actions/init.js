import { InitTemplateError, initTypeSpecProject } from "../../../init/init.js";
import { resolvePath } from "../../path-utils.js";
export async function initAction(host, args) {
    try {
        const outputDir = args.outputDir?.trim();
        const directory = outputDir ? resolvePath(process.cwd(), outputDir) : process.cwd();
        await initTypeSpecProject(host, directory, args);
        return [];
    }
    catch (e) {
        if (e instanceof InitTemplateError) {
            return e.diagnostics;
        }
        throw e;
    }
}
//# sourceMappingURL=init.js.map