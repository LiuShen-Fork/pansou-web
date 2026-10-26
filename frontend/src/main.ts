import { createApp } from 'vue'
import App from './App.vue'

import './assets/styles/pansou.css'
import './assets/styles/admin.css'
import './assets/styles/theme.css'
import { initTheme } from './lib/theme'

// 在挂载前恢复主题，避免首屏闪烁
initTheme()

// 创建应用实例
const app = createApp(App)

// 挂载应用
app.mount('#app')
