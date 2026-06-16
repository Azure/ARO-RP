import { bundle } from "@scalar/json-magic/bundle";
import { mergeObjects } from "../../utils/join/merge-objects.js";
import { upgrade } from "../../utils/upgrade.js";
const getSetIntersection = (a, b) => {
  const result = [];
  for (const value of a) {
    if (b.has(value)) {
      result.push(value);
    }
  }
  return result;
};
const withDefault = (value, defaultValue) => {
  if (Array.isArray(value)) {
    return value.length ? value : defaultValue;
  }
  if (typeof value === "object" && value !== null) {
    return Object.keys(value).length ? value : defaultValue;
  }
  return value ?? defaultValue;
};
const mergePaths = (inputs) => {
  const result = {};
  const conflicts = [];
  for (const paths of inputs) {
    if (typeof paths !== "object") {
      continue;
    }
    for (const [path, pathItem] of Object.entries(paths)) {
      if (!result[path]) {
        result[path] = pathItem;
        continue;
      }
      const intersectingKeys = getSetIntersection(new Set(Object.keys(result[path])), new Set(Object.keys(pathItem)));
      result[path] = { ...result[path], ...pathItem };
      intersectingKeys.forEach((key) => conflicts.push({ method: key, path }));
    }
  }
  return { paths: result, conflicts };
};
const mergeTags = (inputs) => {
  const cache = /* @__PURE__ */ new Set();
  const result = [];
  for (const tags of inputs) {
    for (const tag of tags) {
      if (!cache.has(tag.name)) {
        result.push(tag);
      }
      cache.add(tag.name);
    }
  }
  return result;
};
const mergeServers = (inputs) => {
  const cache = /* @__PURE__ */ new Set();
  const result = [];
  for (const servers of inputs) {
    for (const server of servers) {
      if (!cache.has(server.url)) {
        result.push(server);
      }
      cache.add(server.url);
    }
  }
  return result;
};
const mergeComponents = (inputs) => {
  const result = {};
  const conflicts = [];
  for (const components of inputs) {
    if (typeof components !== "object") {
      continue;
    }
    for (const [key, value] of Object.entries(components)) {
      for (const [name, component] of Object.entries(value)) {
        if (!result[key]) {
          result[key] = {};
        }
        if (result[key][name]) {
          conflicts.push({ componentType: key, name });
        } else {
          result[key][name] = component;
        }
      }
    }
  }
  return { components: result, conflicts };
};
const prefixComponents = async (inputs, prefixes) => {
  for (const index of inputs.keys()) {
    await bundle(inputs[index], {
      treeShake: false,
      urlMap: false,
      plugins: [
        // Plugin to update $ref values to use the prefixed component names
        {
          type: "lifecycle",
          onBeforeNodeProcess: (node) => {
            const ref = node["$ref"];
            if (typeof ref !== "string") {
              return;
            }
            if (!ref.startsWith("#/components/")) {
              return;
            }
            const parts = ref.split("/");
            if (parts.length < 4) {
              return;
            }
            parts[3] = `${prefixes[index] ?? ""}${parts[3]}`;
            node["$ref"] = parts.join("/");
          }
        },
        // Plugin to rename component keys with the prefix
        {
          type: "lifecycle",
          onBeforeNodeProcess: (node, context) => {
            if (context.path.length === 2 && context.path[0] === "components") {
              const prefix = prefixes[index];
              Object.keys(node).forEach((key) => {
                const newKey = `${prefix ?? ""}${key}`;
                const childNode = node[key];
                delete node[key];
                node[newKey] = childNode;
              });
            }
          }
        }
      ]
    });
  }
};
const join = async (inputs, config) => {
  const upgraded = inputs.map((it) => upgrade(it).specification);
  if (config?.prefixComponents) {
    await prefixComponents(upgraded, config.prefixComponents);
  }
  upgraded.reverse();
  const info = upgraded.reduce((acc, curr) => {
    if (curr.info && typeof curr.info === "object") {
      return mergeObjects(acc, curr.info);
    }
    return acc;
  }, {});
  const { paths, conflicts: pathConflicts } = mergePaths(upgraded.map((it) => it.paths ?? {}));
  const { paths: webhooks, conflicts: webhookConflicts } = mergePaths(upgraded.map((it) => it.webhooks ?? {}));
  const tags = mergeTags(upgraded.map((it) => it.tags ?? []));
  const servers = mergeServers(upgraded.map((it) => it.servers ?? []));
  const { components, conflicts: componentConflicts } = mergeComponents(upgraded.map((it) => it.components ?? {}));
  const result = upgraded.reduce((acc, curr) => ({ ...acc, ...curr }), {});
  const conflicts = [
    ...pathConflicts.map((it) => ({ type: "path", ...it })),
    ...webhookConflicts.map((it) => ({ type: "webhook", ...it })),
    ...componentConflicts.map((it) => ({ type: "component", ...it }))
  ];
  if (conflicts.length) {
    return {
      ok: false,
      conflicts
    };
  }
  return {
    ok: true,
    document: {
      ...result,
      info,
      paths,
      webhooks: withDefault(webhooks, void 0),
      tags: withDefault(tags, void 0),
      servers: withDefault(servers, void 0),
      components: withDefault(components, void 0)
    }
  };
};
export {
  join
};
//# sourceMappingURL=join.js.map
