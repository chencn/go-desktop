// 文件职责：配置 Vue、Tailwind、Wails 绑定目录和源码路径别名。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";

// bindingsRoot 保存配置 Vue、Tailwind、Wails 绑定目录和源码路径别名 使用的配置、引用或中间结果。
const bindingsRoot = new URL("./bindings", import.meta.url).pathname
  .replace(/^\/([A-Za-z]:\/)/, "$1")
  .replace(/\\/g, "/");
// srcRoot 保存配置 Vue、Tailwind、Wails 绑定目录和源码路径别名 使用的配置、引用或中间结果。
const srcRoot = new URL("./src", import.meta.url).pathname
  .replace(/^\/([A-Za-z]:\/)/, "$1")
  .replace(/\\/g, "/");

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss(), wails(bindingsRoot)],
  resolve: {
    alias: {
      "@": srcRoot,
    },
  },
  server: {
    host: "127.0.0.1",
  },
});
