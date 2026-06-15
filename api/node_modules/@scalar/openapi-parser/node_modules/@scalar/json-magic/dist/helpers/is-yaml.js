function isYaml(value) {
  return /^\s*(?:-\s*)?[\w\-]+\s*:\s*.+\n.*/.test(value);
}
export {
  isYaml
};
//# sourceMappingURL=is-yaml.js.map
