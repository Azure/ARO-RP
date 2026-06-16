const iterateTitle = (title, checkDuplicates, separator = " #") => {
  if (!checkDuplicates(title)) {
    return title;
  }
  const split = title.split(separator);
  const newTitle = split.length > 1 ? `${split.slice(0, -1).join()}${separator}${Number(split.at(-1)) + 1}` : `${split.join()}${separator}2`;
  return iterateTitle(newTitle, checkDuplicates, separator);
};
export {
  iterateTitle
};
//# sourceMappingURL=iterate-title.js.map
