var _a, _b;
import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
var adminProxyTarget = (_a = process.env.ADMIN_PROXY_TARGET) !== null && _a !== void 0 ? _a : "http://localhost:4870";
var appBasePath = (_b = process.env.APP_BASE_PATH) !== null && _b !== void 0 ? _b : "/panel/";
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
});
