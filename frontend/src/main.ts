// Vue 入口：装配 Pinia、项目级 UI 插件和全局样式后挂载根组件。

import { createApp } from 'vue'
import { createPinia } from 'pinia'
// Element Plus 暗色变量跟随 html.dark 切换；组件样式由 unplugin-vue-components 按需引入。
import 'element-plus/theme-chalk/dark/css-vars.css'
// 命令式组件不经过模板解析，样式需手动引入。
import 'element-plus/es/components/message/style/css'
import App from './App.vue'
import './colors.css'
import './styles.css'
import './styles/layout.css'
import './styles/liquid-glass.css'
// 皮肤层必须排在最后：它要覆盖 App.vue 依赖链里按需引入的 Element Plus 组件样式。
import './styles/element-plus.css'

createApp(App).use(createPinia()).mount('#app')
