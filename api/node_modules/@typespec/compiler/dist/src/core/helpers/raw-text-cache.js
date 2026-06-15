import { SyntaxKind } from "../types.js";
const nodeRawTextCache = new WeakMap();
export function cacheRawText(node, rawText) {
    nodeRawTextCache.set(node, rawText);
}
export function getCachedRawText(node) {
    return nodeRawTextCache.get(node);
}
export function getRawTextWithCache(node) {
    const cached = getCachedRawText(node);
    if (cached !== undefined) {
        return cached;
    }
    let rawText = "";
    if ("rawText" in node) {
        rawText = node.rawText;
    }
    else {
        const scriptNode = getTypeSpecScript(node);
        if (scriptNode) {
            rawText = scriptNode.file.text.slice(node.pos, node.end);
        }
    }
    cacheRawText(node, rawText);
    return rawText;
}
function getTypeSpecScript(node) {
    let current = node;
    while (current.parent) {
        current = current.parent;
    }
    return current.kind === SyntaxKind.TypeSpecScript ? current : undefined;
}
//# sourceMappingURL=raw-text-cache.js.map