/**
 * Redirects the request to a proxy server with a given URL. But not for:
 *
 * - Relative URLs
 * - URLs that seem to point to a local IP (except the proxy is on the same domain)
 * - URLs that don't look like a domain
 **/
export declare const redirectToProxy: (proxyUrl?: string, url?: string) => string;
/**
 * Returns false for requests to localhost, relative URLs, if no proxy is defined …
 **/
export declare const shouldUseProxy: (proxyUrl?: string, url?: string) => url is string;
//# sourceMappingURL=redirect-to-proxy.d.ts.map