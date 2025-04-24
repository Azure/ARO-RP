import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import fs from "fs"

export default defineConfig({
  resolve: {
    alias: [],
  },

  build: {
    emptyOutDir: true,
    target: "edge100",
    sourcemap: true,
    outDir: "../../pkg/portal/assets/v2/build",
    reportCompressedSize: false,
  },

  server: {
    https: {
      key: fs.readFileSync("../../secrets/proxy-client.pem"),
      cert: fs.readFileSync("../../secrets/proxy-client.pem"),
    },
    port: 3000,
    proxy: {
      "/api": {
        target: "https://localhost:8444/",
        changeOrigin: true,
        secure: false,
      },
      "/callback": {
        target: "https://localhost:8444",
        changeOrigin: true,
        secure: false,
      },
      "/subscriptions": {
        target: "https://localhost:8444/subscriptions",
        changeOrigin: true,
        secure: false,
      },
    },
  },

  plugins: [react()],
})
