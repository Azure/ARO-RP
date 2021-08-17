const {createProxyMiddleware} = require("http-proxy-middleware")

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
