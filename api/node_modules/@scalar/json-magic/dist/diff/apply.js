class InvalidChangesDetectedError extends Error {
  constructor(message) {
    super(message);
    this.name = "InvalidChangesDetectedError";
  }
}
const apply = (document, diff) => {
  const applyChange = (current, path, d, depth = 0) => {
    if (path[depth] === void 0) {
      throw new InvalidChangesDetectedError(
        `Process aborted. Path ${path.join(".")} at depth ${depth} is undefined, check diff object`
      );
    }
    if (depth >= path.length - 1) {
      if (d.type === "add" || d.type === "update") {
        current[path[depth]] = d.changes;
      } else {
        if (Array.isArray(current)) {
          current.splice(Number.parseInt(path[depth]), 1);
        } else {
          delete current[path[depth]];
        }
      }
      return;
    }
    if (current[path[depth]] === void 0 || typeof current[path[depth]] !== "object") {
      throw new InvalidChangesDetectedError("Process aborted, check diff object");
    }
    applyChange(current[path[depth]], path, d, depth + 1);
  };
  for (const d of diff) {
    applyChange(document, d.path, d);
  }
  return document;
};
export {
  InvalidChangesDetectedError,
  apply
};
//# sourceMappingURL=apply.js.map
