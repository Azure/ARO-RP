import { Program } from "./program.js";
import { Declaration, Modifier, ModifierFlags } from "./types.js";
/**
 * Checks the modifiers on a declaration node against the allowed and required modifiers.
 *
 * This will report diagnostics in the given program if there are any invalid or missing required modifiers.
 *
 * @param program - The current program (used to report diagnostics).
 * @param node - The declaration node to check.
 * @returns `true` if the modifiers are valid, `false` otherwise.
 */
export declare function checkModifiers(program: Program, node: Declaration): boolean;
export declare function modifiersToFlags(modifiers: Modifier[]): ModifierFlags;
//# sourceMappingURL=modifiers.d.ts.map