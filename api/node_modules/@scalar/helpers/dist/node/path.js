function normalizeArray(parts, allowAboveRoot) {
  let up = 0;
  for (let i = parts.length - 1; i >= 0; i--) {
    const last = parts[i];
    if (last === ".") {
      parts.splice(i, 1);
    } else if (last === "..") {
      parts.splice(i, 1);
      up++;
    } else if (up) {
      parts.splice(i, 1);
      up--;
    }
  }
  if (allowAboveRoot) {
    for (; up--; up) {
      parts.unshift("..");
    }
  }
  return parts;
}
const splitPathRe = /^(\/?|)([\s\S]*?)((?:\.{1,2}|[^/]+?|)(\.[^./]*|))(?:[/]*)$/;
const splitPath = (filename) => splitPathRe.exec(filename).slice(1);
function resolve(...parameters) {
  let resolvedPath = "", resolvedAbsolute = false;
  for (let i = parameters.length - 1; i >= -1 && !resolvedAbsolute; i--) {
    const path2 = i >= 0 ? parameters[i] : "/";
    if (typeof path2 !== "string") {
      throw new TypeError("Arguments to path.resolve must be strings");
    }
    if (!path2) {
      continue;
    }
    resolvedPath = path2 + "/" + resolvedPath;
    resolvedAbsolute = path2.charAt(0) === "/";
  }
  resolvedPath = normalizeArray(
    resolvedPath.split("/").filter((p) => !!p),
    !resolvedAbsolute
  ).join("/");
  return (resolvedAbsolute ? "/" : "") + resolvedPath || ".";
}
function normalize(inputPath) {
  const isPathAbsolute = isAbsolute(inputPath), trailingSlash = inputPath.slice(-1) === "/";
  let path2 = normalizeArray(
    inputPath.split("/").filter((p) => !!p),
    !isPathAbsolute
  ).join("/");
  if (!path2 && !isPathAbsolute) {
    path2 = ".";
  }
  if (path2 && trailingSlash) {
    path2 += "/";
  }
  return (isPathAbsolute ? "/" : "") + path2;
}
function isAbsolute(path2) {
  return path2.charAt(0) === "/";
}
function join(...paths) {
  return normalize(
    paths.filter((p) => {
      if (typeof p !== "string") {
        throw new TypeError("Arguments to path.join must be strings");
      }
      return p;
    }).join("/")
  );
}
function relative(from, to) {
  const fromResolved = resolve(from).substring(1);
  const toResolved = resolve(to).substring(1);
  function trim(arr) {
    let start = 0;
    for (; start < arr.length; start++) {
      if (arr[start] !== "") {
        break;
      }
    }
    let end = arr.length - 1;
    for (; end >= 0; end--) {
      if (arr[end] !== "") {
        break;
      }
    }
    if (start > end) {
      return [];
    }
    return arr.slice(start, end - start + 1);
  }
  const fromParts = trim(fromResolved.split("/"));
  const toParts = trim(toResolved.split("/"));
  const length = Math.min(fromParts.length, toParts.length);
  let samePartsLength = length;
  for (let i = 0; i < length; i++) {
    if (fromParts[i] !== toParts[i]) {
      samePartsLength = i;
      break;
    }
  }
  let outputParts = [];
  for (let i = samePartsLength; i < fromParts.length; i++) {
    outputParts.push("..");
  }
  outputParts = outputParts.concat(toParts.slice(samePartsLength));
  return outputParts.join("/");
}
const sep = "/";
const delimiter = ":";
function dirname(path2) {
  const result = splitPath(path2);
  const root = result[0];
  let dir = result[1];
  if (!root && !dir) {
    return ".";
  }
  if (dir) {
    dir = dir.slice(0, -1);
  }
  return root + dir;
}
function basename(path2, ext) {
  let f = splitPath(path2)[2];
  if (ext && f.slice(-ext.length) === ext) {
    f = f.slice(0, -ext.length);
  }
  return f;
}
function extname(path2) {
  return splitPath(path2)[3];
}
const path = {
  extname,
  basename,
  dirname,
  sep,
  delimiter,
  relative,
  join,
  isAbsolute,
  normalize,
  resolve
};
export {
  basename,
  delimiter,
  dirname,
  extname,
  isAbsolute,
  join,
  normalize,
  path,
  relative,
  resolve,
  sep
};
//# sourceMappingURL=path.js.map
