/**
 * This function converts an object to XML.
 * Values are automatically escaped to prevent XML injection attacks.
 */
export declare function json2xml(data: Record<string, any>, options?: {
    indent?: string;
    format?: boolean;
    xmlDeclaration?: boolean;
}): string;
//# sourceMappingURL=json2xml.d.ts.map