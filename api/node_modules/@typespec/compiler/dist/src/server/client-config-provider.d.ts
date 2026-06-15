import { Connection } from "vscode-languageserver/node.js";
import { ServerHost } from "./types.js";
interface LSPConfig {
    emit?: string[];
}
interface Config {
    lsp?: LSPConfig;
    entrypoint?: string[];
}
/**
 * TypeSpec client-side configuration provider
 * Inspired by VS Code's WorkspaceConfiguration API for extensibility
 */
export interface ClientConfigProvider {
    /**
     * Initialize client configuration with connection and host
     * @param connection Language server connection
     * @param host Server host instance
     */
    initialize(connection: Connection, host: ServerHost): Promise<void>;
    config?: Config;
}
export declare function createClientConfigProvider(): ClientConfigProvider;
export {};
//# sourceMappingURL=client-config-provider.d.ts.map