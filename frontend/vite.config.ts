// 文件职责：配置 Vue、Tailwind、Element Plus 按需注册、Wails 绑定目录和源码路径别名。

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

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
  plugins: [
    vue(),
    tailwindcss(),
    // Element Plus 按需注册：模板里的 el-* 自动带样式引入；命令式组件样式在 main.ts 手动补齐。
    Components({ dts: "src/components.d.ts", resolvers: [ElementPlusResolver()] }),
    wails(bindingsRoot),
  ],
  resolve: {
    alias: {
      "@": srcRoot,
    },
  },
  server: {
    host: "127.0.0.1",
  },
  build: {
    // 桌面端本地内嵌静态资源，放宽 Web 默认的 500 kB 警报阈值至 1000 kB
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules")) {
            // 将纯底层框架运行时（Vue 核心与 Pinia）拆为独立 chunk
            if (id.includes("vue") && !id.includes("element-plus") && !id.includes("lucide")) {
              return "vendor-vue";
            }
          }
        },
      },
    },
  },
});
