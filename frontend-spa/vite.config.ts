import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  envDir: "../",
  plugins: [tailwindcss(), reactRouter()],
  resolve: {
    tsconfigPaths: true,
    dedupe: ["react", "react-dom"],
  },
  server: {
    port: 3000,
  },
  optimizeDeps: {
    include: ["react", "react-dom", "react-router"],
  },
});
