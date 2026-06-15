const convertToLocalRef = (ref, currentContext, schemas) => {
  const [baseUrl, pathOrAnchor] = ref.split("#", 2);
  if (baseUrl) {
    if (!schemas.has(baseUrl)) {
      return void 0;
    }
    if (!pathOrAnchor) {
      return schemas.get(baseUrl);
    }
    if (pathOrAnchor.startsWith("/")) {
      return `${schemas.get(baseUrl)}${pathOrAnchor}`;
    }
    return schemas.get(`${baseUrl}#${pathOrAnchor}`);
  }
  if (pathOrAnchor) {
    if (pathOrAnchor.startsWith("/")) {
      return pathOrAnchor.slice(1);
    }
    return schemas.get(`${currentContext}#${pathOrAnchor}`);
  }
  return void 0;
};
export {
  convertToLocalRef
};
//# sourceMappingURL=convert-to-local-ref.js.map
