const webpack = require("webpack")

module.exports = function override(config) {
  const fallback = config.resolve.fallback || {}

  Object.assign(fallback, {
    path: require.resolve("path-browserify"),
    buffer: require.resolve("buffer/"),
  })

  config.resolve.fallback = fallback

  config.module.rules.push({
    test: /\.m?js/,
    resolve: {
      fullySpecified: false,
    },
  })

  config.plugins = (config.plugins || []).concat([
    new webpack.ProvidePlugin({
      process: "process/browser",
      Buffer: ["buffer", "Buffer"],
    }),
  ])

  return config
}
