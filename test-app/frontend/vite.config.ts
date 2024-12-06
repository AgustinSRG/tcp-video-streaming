import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

import dotenv from "dotenv";
dotenv.config();

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [vue()],
    server: {
        port: parseInt(process.env.DEV_PORT || "8080", 10) || 8080,
    },
    build: {
        assetsInlineLimit: 0,
        sourcemap: true,
    },
    resolve: {
        alias: {
            "@": fileURLToPath(new URL("./src", import.meta.url)),
        },
    },
});
