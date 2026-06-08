// 文件职责：main.ts 中的业务流程、状态和数据结构。
// 说明：注释覆盖文件、类型、方法和关键变量；代码执行路径保持不变。

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { uiPlugin } from './shared/ui/plugin'
import './styles.css'
import './styles/layout.css'
import './styles/antd-scheme.css'

createApp(App).use(createPinia()).use(uiPlugin).mount('#app')
