import { isMacOS } from "./is-mac-os.js";
const hasModifier = (keydown) => {
  const modifier = isMacOS() ? "metaKey" : "ctrlKey";
  return keydown[modifier];
};
export {
  hasModifier
};
//# sourceMappingURL=has-modifier.js.map
