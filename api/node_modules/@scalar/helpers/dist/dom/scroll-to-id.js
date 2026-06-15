const scrollToId = (id, focus) => {
  const scrollToElement = (element2) => {
    element2.scrollIntoView();
    if (focus) {
      element2.focus();
    }
  };
  const element = document.getElementById(id);
  if (element) {
    scrollToElement(element);
    return;
  }
  const stopTime = Date.now() + 1e3;
  const tryScroll = () => {
    const element2 = document.getElementById(id);
    if (element2) {
      scrollToElement(element2);
      return;
    }
    if (Date.now() < stopTime) {
      requestAnimationFrame(tryScroll);
    }
  };
  requestAnimationFrame(tryScroll);
};
export {
  scrollToId
};
//# sourceMappingURL=scroll-to-id.js.map
