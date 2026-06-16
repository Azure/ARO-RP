const REQUEST_METHODS = {
  get: {
    short: "GET",
    colorClass: "text-blue",
    colorVar: "var(--scalar-color-blue)",
    backgroundColor: "bg-blue/10"
  },
  post: {
    short: "POST",
    colorClass: "text-green",
    colorVar: "var(--scalar-color-green)",
    backgroundColor: "bg-green/10"
  },
  put: {
    short: "PUT",
    colorClass: "text-orange",
    colorVar: "var(--scalar-color-orange)",
    backgroundColor: "bg-orange/10"
  },
  patch: {
    short: "PATCH",
    colorClass: "text-yellow",
    colorVar: "var(--scalar-color-yellow)",
    backgroundColor: "bg-yellow/10"
  },
  delete: {
    short: "DEL",
    colorClass: "text-red",
    colorVar: "var(--scalar-color-red)",
    backgroundColor: "bg-red/10"
  },
  options: {
    short: "OPTS",
    colorClass: "text-purple",
    colorVar: "var(--scalar-color-purple)",
    backgroundColor: "bg-purple/10"
  },
  head: {
    short: "HEAD",
    colorClass: "text-c-2",
    colorVar: "var(--scalar-color-2)",
    backgroundColor: "bg-c-2/10"
  },
  trace: {
    short: "TRACE",
    colorClass: "text-c-2",
    colorVar: "var(--scalar-color-2)",
    backgroundColor: "bg-c-2/10"
  }
};
const getHttpMethodInfo = (methodName) => {
  const normalizedMethod = methodName.trim().toLowerCase();
  return REQUEST_METHODS[normalizedMethod] ?? {
    short: normalizedMethod,
    color: "text-c-2",
    backgroundColor: "bg-c-2"
  };
};
export {
  REQUEST_METHODS,
  getHttpMethodInfo
};
//# sourceMappingURL=http-info.js.map
