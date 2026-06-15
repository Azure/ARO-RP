import { deepClone } from "../utils/misc.js";
export function createClientConfigProvider() {
    let config;
    async function initialize(connection, host) {
        try {
            const configs = await connection.workspace.getConfiguration("typespec");
            // Transform the raw configuration to match our Config interface
            config = deepClone(configs);
            host.log({ level: "debug", message: "vscode settings loaded", detail: config });
            connection.onDidChangeConfiguration(async (params) => {
                if (params.settings) {
                    const newConfigs = params.settings?.typespec;
                    config = deepClone(newConfigs);
                }
                host.log({ level: "debug", message: "Configuration changed", detail: params.settings });
            });
        }
        catch (error) {
            host.log({
                level: "error",
                message: "An error occurred while loading the VSCode settings",
                detail: error,
            });
        }
    }
    return {
        initialize,
        get config() {
            return config;
        },
    };
}
//# sourceMappingURL=client-config-provider.js.map