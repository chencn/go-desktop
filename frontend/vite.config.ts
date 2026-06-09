// 文件职责：配置 Vue、Tailwind、Wails 绑定目录和源码路径别名。

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";

// Wails Vite 插件要求传入可跨平台识别的 bindings 目录，Windows 下去掉 URL pathname 的前导斜杠。
const bindingsRoot = new URL("./bindings", import.meta.url).pathname
  .replace(/^\/([A-Za-z]:\/)/, "$1")
  .replace(/\\/g, "/");
// @ 别名指向 frontend/src，供页面、store 和共享工具使用同一套导入路径。
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
