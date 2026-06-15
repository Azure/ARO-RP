import { TextDocumentChangeEvent, TextDocumentIdentifier } from "vscode-languageserver";
import { TextDocument } from "vscode-languageserver-textdocument";
import { ServerHost } from "./types.js";
/**
 * Service managing files in the language server.
 */
export interface FileService {
    upToDate(document: TextDocument | TextDocumentIdentifier): boolean;
    fileURLToRealPath(url: string): Promise<string>;
    getPath(document: TextDocument | TextDocumentIdentifier): Promise<string>;
    getOpenDocument(path: string): TextDocument | undefined;
    getURL(path: string): string;
    getOpenDocumentInitVersion(uri: string): number | undefined;
    notifyDocumentOpened(arg: TextDocumentChangeEvent<TextDocument>): void;
    notifyDocumentClosed(arg: TextDocumentChangeEvent<TextDocument>): void;
}
export interface FileServiceOptions {
    serverHost: ServerHost;
}
export declare function createFileService({ serverHost }: FileServiceOptions): FileService;
//# sourceMappingURL=file-service.d.ts.map