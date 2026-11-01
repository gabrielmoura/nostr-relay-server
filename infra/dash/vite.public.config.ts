import path from "node:path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

// This build intentionally contains only the public entrypoint. It is embedded
// in the relay listener and must never import the admin router or Apollo client.
export default defineConfig({
  base: "/",
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "public-dist",
    emptyOutDir: true,
    rollupOptions: { input: path.resolve(__dirname, "public.html") },
  },
  resolve: { alias: { "@": path.resolve(__dirname, "./src") } },
})
