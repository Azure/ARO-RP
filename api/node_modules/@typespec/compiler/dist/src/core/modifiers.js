// Copyright (c) Microsoft Corporation
// Licensed under the MIT License.
import { compilerAssert } from "./diagnostics.js";
import { createDiagnostic } from "./messages.js";
import { SyntaxKind } from "./types.js";
/**
 * The default compatibility for all declaration syntax nodes.
 *
 * By default, only the `internal` modifier is allowed on all declaration syntax nodes.
 * No modifiers are required by default.
 */
const DEFAULT_COMPATIBILITY = {
    allowed: 4 /* ModifierFlags.Internal */,
    required: 0 /* ModifierFlags.None */,
};
const NO_MODIFIERS = {
    allowed: 0 /* ModifierFlags.None */,
    required: 0 /* ModifierFlags.None */,
};
const SYNTAX_MODIFIERS = {
    [SyntaxKind.NamespaceStatement]: NO_MODIFIERS,
    [SyntaxKind.OperationStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.ModelStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.ScalarStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.InterfaceStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.UnionStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.EnumStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.AliasStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.ConstStatement]: DEFAULT_COMPATIBILITY,
    [SyntaxKind.DecoratorDeclarationStatement]: {
        allowed: 6 /* ModifierFlags.All */,
        required: 2 /* ModifierFlags.Extern */,
    },
    [SyntaxKind.FunctionDeclarationStatement]: {
        allowed: 6 /* ModifierFlags.All */,
        required: 2 /* ModifierFlags.Extern */,
    },
};
/**
 * Checks the modifiers on a declaration node against the allowed and required modifiers.
 *
 * This will report diagnostics in the given program if there are any invalid or missing required modifiers.
 *
 * @param program - The current program (used to report diagnostics).
 * @param node - The declaration node to check.
 * @returns `true` if the modifiers are valid, `false` otherwise.
 */
export function checkModifiers(program, node) {
    const compatibility = SYNTAX_MODIFIERS[node.kind];
    let isValid = true;
    // Emit experimental warning for any use of the 'internal' modifier.
    if (node.modifierFlags & 4 /* ModifierFlags.Internal */) {
        const internalModifiers = filterModifiersByFlags(node.modifiers, 4 /* ModifierFlags.Internal */);
        for (const modifier of internalModifiers) {
            program.reportDiagnostic(createDiagnostic({
                code: "experimental-feature",
                messageId: "internal",
                target: modifier,
            }));
        }
    }
    const invalidModifiers = node.modifierFlags & ~compatibility.allowed;
    if (invalidModifiers) {
        // There is at least one modifier used that is not allowed on this syntax node.
        isValid = false;
        const invalidModifierList = filterModifiersByFlags(node.modifiers, invalidModifiers);
        for (const modifier of invalidModifierList) {
            const modifierText = getTextForModifier(modifier);
            program.reportDiagnostic(createDiagnostic({
                code: "invalid-modifier",
                messageId: "not-allowed",
                format: { modifier: modifierText, nodeKind: getDeclarationKindText(node.kind) },
                target: modifier,
            }));
        }
    }
    const missingRequiredModifiers = compatibility.required & ~node.modifierFlags;
    if (missingRequiredModifiers) {
        // There is at least one required modifier missing from this syntax node.
        isValid = false;
        for (const missing of getNamesOfModifierFlags(missingRequiredModifiers)) {
            program.reportDiagnostic(createDiagnostic({
                code: "invalid-modifier",
                messageId: "missing-required",
                format: { modifier: missing, nodeKind: getDeclarationKindText(node.kind) },
                target: node,
            }));
        }
    }
    return isValid;
}
function filterModifiersByFlags(modifiers, flags) {
    const result = [];
    for (const modifier of modifiers) {
        if (modifierToFlag(modifier) & flags) {
            result.push(modifier);
        }
    }
    return result;
}
export function modifiersToFlags(modifiers) {
    let flags = 0 /* ModifierFlags.None */;
    for (const modifier of modifiers) {
        flags |= modifierToFlag(modifier);
    }
    return flags;
}
function modifierToFlag(modifier) {
    switch (modifier.kind) {
        case SyntaxKind.ExternKeyword:
            return 2 /* ModifierFlags.Extern */;
        case SyntaxKind.InternalKeyword:
            return 4 /* ModifierFlags.Internal */;
        default:
            compilerAssert(false, `Unknown modifier kind: ${modifier.kind}`);
    }
}
function getTextForModifier(modifier) {
    switch (modifier.kind) {
        case SyntaxKind.ExternKeyword:
            return "extern";
        case SyntaxKind.InternalKeyword:
            return "internal";
        default:
            compilerAssert(false, `Unknown modifier kind: ${modifier.kind}`);
    }
}
function getNamesOfModifierFlags(flags) {
    const names = [];
    if (flags & 2 /* ModifierFlags.Extern */) {
        names.push("extern");
    }
    if (flags & 4 /* ModifierFlags.Internal */) {
        names.push("internal");
    }
    return names;
}
function getDeclarationKindText(nodeKind) {
    switch (nodeKind) {
        case SyntaxKind.NamespaceStatement:
            return "namespace";
        case SyntaxKind.OperationStatement:
            return "op";
        case SyntaxKind.ModelStatement:
            return "model";
        case SyntaxKind.ScalarStatement:
            return "scalar";
        case SyntaxKind.InterfaceStatement:
            return "interface";
        case SyntaxKind.UnionStatement:
            return "union";
        case SyntaxKind.EnumStatement:
            return "enum";
        case SyntaxKind.AliasStatement:
            return "alias";
        case SyntaxKind.DecoratorDeclarationStatement:
            return "dec";
        case SyntaxKind.FunctionDeclarationStatement:
            return "function";
        case SyntaxKind.ConstStatement:
            return "const";
        default:
            compilerAssert(false, `Unknown declaration kind: ${nodeKind}`);
    }
}
//# sourceMappingURL=modifiers.js.map