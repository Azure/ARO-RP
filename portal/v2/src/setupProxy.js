const { createProxyMiddleware } = require("http-proxy-middleware")

// These proxies are necessary for the webpack frontend development server to function correctly when logging into Azure.

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
    "/subscriptions",
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
