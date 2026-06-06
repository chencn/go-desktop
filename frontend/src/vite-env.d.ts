// 文件职责：vite-env.d.ts 中的业务流程、状态和数据结构。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  // component 保存 vite-env.d.ts 中的业务流程、状态和数据结构 使用的配置、引用或中间结果。
  const component: DefineComponent<object, object, unknown>
  // default export 暴露 vite-env.d.ts 中的业务流程、状态和数据结构 的模块配置或组件定义，供构建工具加载。
  export default component
}
