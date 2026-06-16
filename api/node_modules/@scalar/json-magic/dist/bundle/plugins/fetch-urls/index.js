import { isRemoteUrl } from "../../../bundle/bundle.js";
import { createLimiter } from "../../../bundle/create-limiter.js";
import { normalize } from "../../../helpers/normalize.js";
const getHost = (url) => {
  try {
    return new URL(url).host;
  } catch {
    return null;
  }
};
async function fetchUrl(url, limiter, config) {
  try {
    const host = getHost(url);
    const headers = config?.headers?.find((a) => a.domains.find((d) => d === host) !== void 0)?.headers;
    const exec = config?.fetch ?? fetch;
    const result = await limiter(
      () => exec(url, {
        headers
      })
    );
    if (result.ok) {
      const body = await result.text();
      return {
        ok: true,
        data: normalize(body),
        raw: body
      };
    }
    const contentType = result.headers.get("Content-Type") ?? "";
    if (["text/html", "application/xml"].includes(contentType)) {
      console.warn(`[WARN] We only support JSON/YAML formats, received ${contentType}`);
    }
    console.warn(`[WARN] Fetch failed with status ${result.status} ${result.statusText} for URL: ${url}`);
    return {
      ok: false
    };
  } catch {
    console.warn(`[WARN] Failed to parse JSON/YAML from URL: ${url}`);
    return {
      ok: false
    };
  }
}
function fetchUrls(config) {
  const limiter = config?.limit ? createLimiter(config.limit) : (fn) => fn();
  return {
    type: "loader",
    validate: isRemoteUrl,
    exec: (value) => fetchUrl(value, limiter, config)
  };
}
export {
  fetchUrl,
  fetchUrls
};
//# sourceMappingURL=index.js.map
