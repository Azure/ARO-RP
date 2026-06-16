function migrateXmlObjects(obj) {
  if (obj === null || typeof obj !== "object") {
    return;
  }
  if (Array.isArray(obj)) {
    for (const item of obj) {
      migrateXmlObjects(item);
    }
    return;
  }
  if (obj.xml && typeof obj.xml === "object") {
    if (obj.xml.wrapped === true && obj.xml.attribute === true) {
      throw new Error("Invalid XML configuration: wrapped and attribute cannot be true at the same time.");
    }
    if (obj.xml.wrapped === true) {
      delete obj.xml.wrapped;
      obj.xml.nodeType = "element";
    }
    if (obj.xml.attribute === true) {
      delete obj.xml.attribute;
      obj.xml.nodeType = "attribute";
    }
  }
  for (const key in obj) {
    if (Object.hasOwn(obj, key)) {
      migrateXmlObjects(obj[key]);
    }
  }
}
function migrateTagGroups(document) {
  if (document["x-tagGroups"] && Array.isArray(document["x-tagGroups"])) {
    const tagGroups = document["x-tagGroups"];
    if (!document.tags) {
      document.tags = [];
    }
    const tagGroupMap = /* @__PURE__ */ new Map();
    for (const group of tagGroups) {
      for (const tagName of group.tags) {
        tagGroupMap.set(tagName, group.name);
      }
    }
    if (Array.isArray(document.tags)) {
      for (const tag of document.tags) {
        if (typeof tag === "object" && tag !== null && "name" in tag) {
          const groupName = tagGroupMap.get(tag.name);
          if (groupName) {
            if (groupName.toLowerCase().includes("nav") || groupName.toLowerCase().includes("navigation")) {
              tag.kind = "nav";
            } else if (groupName.toLowerCase().includes("audience")) {
              tag.kind = "audience";
            } else if (groupName.toLowerCase().includes("badge")) {
              tag.kind = "badge";
            } else {
              tag.kind = "nav";
            }
          }
        }
      }
    }
    delete document["x-tagGroups"];
  }
}
function upgradeFromThreeOneToThreeTwo(originalDocument) {
  const document = originalDocument;
  if (document !== null && typeof document === "object" && typeof document.openapi === "string" && document.openapi?.startsWith("3.1")) {
    document.openapi = "3.2.0";
  } else {
    return document;
  }
  migrateTagGroups(document);
  migrateXmlObjects(document);
  return document;
}
export {
  upgradeFromThreeOneToThreeTwo
};
//# sourceMappingURL=upgrade-from-three-one-to-three-two.js.map
