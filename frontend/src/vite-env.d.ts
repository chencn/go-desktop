// 文件职责：补充 Vite 客户端类型和 .vue 单文件组件模块声明。

/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  // component 让 TypeScript 把 .vue 文件识别为 Vue 组件模块，具体 props/emit 由组件自身声明。
  const component: DefineComponent<object, object, unknown>
  export default component
}
