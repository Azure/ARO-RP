const freezeElement = (element) => {
  if (!element) {
    return () => null;
  }
  const rect = element.getBoundingClientRect();
  const initialViewportTop = rect.top;
  let rafId = null;
  const observer = new MutationObserver((mutations) => {
    const shouldProcess = mutations.some(
      (mutation) => mutation.type === "childList" || mutation.type === "attributes" && (mutation.attributeName === "style" || mutation.attributeName === "class")
    );
    if (!shouldProcess) {
      return;
    }
    if (rafId !== null) {
      cancelAnimationFrame(rafId);
    }
    rafId = requestAnimationFrame(() => {
      const newRect = element.getBoundingClientRect();
      const currentViewportTop = newRect.top;
      if (currentViewportTop !== initialViewportTop) {
        const diff = currentViewportTop - initialViewportTop;
        window.scrollBy(0, diff);
      }
      rafId = null;
    });
  });
  observer.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ["style", "class"],
    characterData: false
  });
  return () => {
    if (rafId !== null) {
      cancelAnimationFrame(rafId);
    }
    observer.disconnect();
  };
};
export {
  freezeElement
};
//# sourceMappingURL=freeze-element.js.map
