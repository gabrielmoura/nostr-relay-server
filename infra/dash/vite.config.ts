import path from "node:path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const adminProxyTarget = process.env.ADMIN_PROXY_TARGET ?? "http://localhost:4870"
const appBasePath = process.env.APP_BASE_PATH ?? "/panel/"

export default defineConfig({
  base: appBasePath,
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/admin": {
        target: adminProxyTarget,
        changeOrigin: true,
      },
      "/metrics": {
        target: adminProxyTarget,
        changeOrigin: true,
      },
    },
  },
})
