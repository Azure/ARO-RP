import { isLocalUrl } from "./is-local-url.js";
import { isRelativePath } from "./is-relative-path.js";
const redirectToProxy = (proxyUrl, url) => {
  try {
    if (!shouldUseProxy(proxyUrl, url)) {
      return url ?? "";
    }
    const newUrl = new URL(url);
    const temporaryProxyUrl = isRelativePath(proxyUrl) ? `http://localhost${proxyUrl}` : proxyUrl;
    newUrl.href = temporaryProxyUrl;
    newUrl.searchParams.append("scalar_url", url);
    const result = isRelativePath(proxyUrl) ? newUrl.toString().replace(/^http:\/\/localhost/, "") : newUrl.toString();
    return result;
  } catch {
    return url ?? "";
  }
};
const shouldUseProxy = (proxyUrl, url) => {
  try {
    if (!proxyUrl || !url) {
      return false;
    }
    if (isRelativePath(url)) {
      return false;
    }
    if (isRelativePath(proxyUrl)) {
      return true;
    }
    if (isLocalUrl(proxyUrl)) {
      return true;
    }
    if (isLocalUrl(url)) {
      return false;
    }
    return true;
  } catch {
    return false;
  }
};
export {
  redirectToProxy,
  shouldUseProxy
};
//# sourceMappingURL=redirect-to-proxy.js.map
