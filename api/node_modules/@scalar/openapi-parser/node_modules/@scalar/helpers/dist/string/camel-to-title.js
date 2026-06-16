const camelToTitleWords = (camelStr = "") => camelStr.replace(/([A-Z]{2,})/g, " $1").replace(/([A-Z])(?=[a-z])/g, " $1").replace(/^./, (str) => str.toUpperCase()).trim();
export {
  camelToTitleWords
};
//# sourceMappingURL=camel-to-title.js.map
