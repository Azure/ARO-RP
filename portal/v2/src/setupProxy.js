const { createProxyMiddleware } = require("http-proxy-middleware")

// These proxies are necessary for the webpack frontend development server to function correctly when logging into Azure.
const proxiedClusterPaths = [
  "/subscriptions/**/resourcegroups/**/providers/microsoft.redhatopenshift/openshiftclusters/**/prometheus",
  "/subscriptions/**/resourcegroups/**/providers/microsoft.redhatopenshift/openshiftclusters/**/kubeconfig",
  "/subscriptions/**/resourcegroups/**/providers/microsoft.redhatopenshift/openshiftclusters/**/ssh",
]

module.exports = function (app) {
  app.use(
    "/api",
    createProxyMiddleware({
      target: "https://localhost:8444",
      changeOrigin: true,
      secure: false,
    })
  )
  app.use(
    proxiedClusterPaths,
    createProxyMiddleware({
      target: "https://localhost:8444",
      changeOrigin: true,
      secure: false,
    })
  )
  app.use(
    "/callback",
    createProxyMiddleware({
      target: "https://localhost:8444",
      changeOrigin: true,
      secure: false,
    })
  )
}
