// Vue 入口：装配 Pinia、项目级 UI 插件和全局样式后挂载根组件。

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { uiPlugin } from './shared/ui/plugin'
import './styles.css'
import './styles/layout.css'
import './styles/artistic-scheme.css'

createApp(App).use(createPinia()).use(uiPlugin).mount('#app')
