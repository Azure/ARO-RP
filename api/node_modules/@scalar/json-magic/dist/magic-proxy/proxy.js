import { isObject } from "@scalar/helpers/object/is-object";
import { convertToLocalRef } from "../helpers/convert-to-local-ref.js";
import { getId, getSchemas } from "../helpers/get-schemas.js";
import { getValueByPath } from "../helpers/get-value-by-path.js";
import { createPathFromSegments, parseJsonPointer } from "../helpers/json-path-utils.js";
const isMagicProxy = Symbol("isMagicProxy");
const magicProxyTarget = Symbol("magicProxyTarget");
const REF_VALUE = "$ref-value";
const REF_KEY = "$ref";
const createMagicProxy = (target, options, args = {
  root: target,
  proxyCache: /* @__PURE__ */ new WeakMap(),
  cache: /* @__PURE__ */ new Map(),
  schemas: getSchemas(target),
  currentContext: ""
}) => {
  if (!isObject(target) && !Array.isArray(target)) {
    return target;
  }
  if (args.proxyCache.has(target)) {
    return args.proxyCache.get(target);
  }
  const handler = {
    /**
     * Proxy "get" trap for magic proxy.
     * - If accessing the special isMagicProxy symbol, return true to identify proxy.
     * - If accessing the magicProxyTarget symbol, return the original target object.
     * - Hide properties starting with __scalar_ by returning undefined.
     * - If accessing "$ref-value" and the object has a local $ref, resolve and return the referenced value as a new magic proxy.
     * - For all other properties, recursively wrap the returned value in a magic proxy (if applicable).
     */
    get(target2, prop, receiver) {
      if (prop === isMagicProxy) {
        return true;
      }
      if (prop === magicProxyTarget) {
        return target2;
      }
      if (typeof prop === "string" && prop.startsWith("__scalar_") && !options?.showInternal) {
        return void 0;
      }
      const ref = Reflect.get(target2, REF_KEY, receiver);
      const id = getId(target2);
      if (prop === REF_VALUE && typeof ref === "string") {
        if (args.cache.has(ref)) {
          return args.cache.get(ref);
        }
        const path = convertToLocalRef(ref, id ?? args.currentContext, args.schemas);
        if (path === void 0) {
          return void 0;
        }
        const resolvedValue = getValueByPath(args.root, parseJsonPointer(`#/${path}`));
        if (isMagicProxyObject(resolvedValue.value)) {
          return resolvedValue.value;
        }
        const proxiedValue = createMagicProxy(resolvedValue.value, options, {
          ...args,
          currentContext: resolvedValue.context
        });
        args.cache.set(ref, proxiedValue);
        return proxiedValue;
      }
      const value = Reflect.get(target2, prop, receiver);
      if (isMagicProxyObject(value)) {
        return value;
      }
      return createMagicProxy(value, options, { ...args, currentContext: id ?? args.currentContext });
    },
    /**
     * Proxy "set" trap for magic proxy.
     * Allows setting properties on the proxied object.
     * This will update the underlying target object.
     *
     * Note: it will not update if the property starts with __scalar_
     * Those will be considered private properties by the proxy
     */
    set(target2, prop, newValue, receiver) {
      const ref = Reflect.get(target2, REF_KEY, receiver);
      if (typeof prop === "string" && prop.startsWith("__scalar_") && !options?.showInternal) {
        return true;
      }
      if (prop === REF_VALUE && typeof ref === "string") {
        const id = getId(target2);
        const path = convertToLocalRef(ref, id ?? args.currentContext, args.schemas);
        if (path === void 0) {
          return void 0;
        }
        const segments = parseJsonPointer(`#/${path}`);
        if (segments.length === 0) {
          return false;
        }
        const getParentNode = () => getValueByPath(args.root, segments.slice(0, -1)).value;
        if (getParentNode() === void 0) {
          createPathFromSegments(args.root, segments.slice(0, -1));
          console.warn(
            `Trying to set $ref-value for invalid reference: ${ref}

Please fix your input file to fix this issue.`
          );
        }
        getParentNode()[segments.at(-1)] = newValue;
        return true;
      }
      return Reflect.set(target2, prop, newValue, receiver);
    },
    /**
     * Proxy "deleteProperty" trap for magic proxy.
     * Allows deleting properties from the proxied object.
     * This will update the underlying target object.
     */
    deleteProperty(target2, prop) {
      return Reflect.deleteProperty(target2, prop);
    },
    /**
     * Proxy "has" trap for magic proxy.
     * - Pretend that "$ref-value" exists if "$ref" exists on the target.
     *   This allows expressions like `"$ref-value" in obj` to return true for objects with a $ref,
     *   even though "$ref-value" is a virtual property provided by the proxy.
     * - Hide properties starting with __scalar_ by returning false.
     * - For all other properties, defer to the default Reflect.has behavior.
     */
    has(target2, prop) {
      if (typeof prop === "string" && prop.startsWith("__scalar_") && !options?.showInternal) {
        return false;
      }
      if (prop === REF_VALUE && REF_KEY in target2) {
        return true;
      }
      return Reflect.has(target2, prop);
    },
    /**
     * Proxy "ownKeys" trap for magic proxy.
     * - Returns the list of own property keys for the proxied object.
     * - If the object has a "$ref" property, ensures that "$ref-value" is also included in the keys,
     *   even though "$ref-value" is a virtual property provided by the proxy.
     *   This allows Object.keys, Reflect.ownKeys, etc. to include "$ref-value" for objects with $ref.
     * - Filters out properties starting with __scalar_.
     */
    ownKeys(target2) {
      const keys = Reflect.ownKeys(target2);
      const filteredKeys = keys.filter(
        (key) => typeof key !== "string" || !(key.startsWith("__scalar_") && !options?.showInternal)
      );
      if (REF_KEY in target2 && !filteredKeys.includes(REF_VALUE)) {
        filteredKeys.push(REF_VALUE);
      }
      return filteredKeys;
    },
    /**
     * Proxy "getOwnPropertyDescriptor" trap for magic proxy.
     * - For the virtual "$ref-value" property, returns a descriptor that makes it appear as a regular property.
     * - Hide properties starting with __scalar_ by returning undefined.
     * - For all other properties, delegates to the default Reflect.getOwnPropertyDescriptor behavior.
     * - This ensures that Object.getOwnPropertyDescriptor and similar methods work correctly with the virtual property.
     */
    getOwnPropertyDescriptor(target2, prop) {
      if (typeof prop === "string" && prop.startsWith("__scalar_") && !options?.showInternal) {
        return void 0;
      }
      const ref = Reflect.get(target2, REF_KEY);
      if (prop === REF_VALUE && typeof ref === "string") {
        return {
          configurable: true,
          enumerable: true,
          value: void 0,
          writable: false
        };
      }
      return Reflect.getOwnPropertyDescriptor(target2, prop);
    }
  };
  const proxied = new Proxy(target, handler);
  args.proxyCache.set(target, proxied);
  return proxied;
};
const isMagicProxyObject = (obj) => {
  return typeof obj === "object" && obj !== null && obj[isMagicProxy] === true;
};
function getRaw(obj) {
  if (typeof obj !== "object" || obj === null) {
    return obj;
  }
  if (obj[isMagicProxy]) {
    return obj[magicProxyTarget];
  }
  return obj;
}
export {
  createMagicProxy,
  getRaw
};
//# sourceMappingURL=proxy.js.map
