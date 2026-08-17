import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './styles/tokens.css'
import './styles/base.css'
import './styles/layout.css'
import './styles/components.css'
import './styles/views.css'
import './styles/admin.css'
import router from './router'
import { initTheme } from './stores/theme'

const pinia = createPinia()
initTheme(pinia)
createApp(App).use(pinia).use(router).mount('#app')