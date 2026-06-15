/**
 * Scroll Freezing Utility
 * "Freezes" the scroll position of an element, so that it doesn't move when the rest of the content changes
 *
 * @example
 * const unfreeze = freezeElement(document.querySelector('#your-element'))
 * ... content changes ...
 * unfreeze()
 */
export declare const freezeElement: (element: HTMLElement) => () => void;
//# sourceMappingURL=freeze-element.d.ts.map