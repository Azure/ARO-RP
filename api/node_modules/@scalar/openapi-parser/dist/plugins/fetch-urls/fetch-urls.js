const fetchUrlsDefaultConfiguration = {
  limit: 20
};
const fetchUrls = (customConfiguration) => {
  let numberOfRequests = 0;
  const configuration = {
    ...fetchUrlsDefaultConfiguration,
    ...customConfiguration
  };
  return {
    check(value) {
      if (typeof value !== "string") {
        return false;
      }
      if (!value.startsWith("http://") && !value.startsWith("https://")) {
        return false;
      }
      return true;
    },
    async get(value) {
      if (configuration?.limit !== false && numberOfRequests >= configuration?.limit) {
        console.warn(`[fetchUrls] Maximum number of requests reeached (${configuration?.limit}), skipping request`);
        return void 0;
      }
      try {
        numberOfRequests++;
        const response = await (configuration?.fetch ? configuration.fetch(value) : fetch(value));
        return await response.text();
      } catch (error) {
        console.error("[fetchUrls]", error.message, `(${value})`);
        return void 0;
      }
    }
  };
};
export {
  fetchUrls,
  fetchUrlsDefaultConfiguration
};
//# sourceMappingURL=fetch-urls.js.map
