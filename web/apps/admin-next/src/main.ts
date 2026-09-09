import { createApp } from 'vue'
import {
  Alert,
  Avatar,
  Button,
  Card,
  Checkbox,
  Descriptions,
  Divider,
  Dropdown,
  Empty,
  Form,
  Grid,
  Input,
  InputNumber,
  Layout,
  Menu,
  Modal,
  Popconfirm,
  Radio,
  Result,
  Select,
  Space,
  Spin,
  Statistic,
  Switch,
  Table,
  Tabs,
  Tag,
  Textarea,
  Tooltip
} from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'

import App from '@/App.vue'
import router from '@/router'
import { createPinia } from 'pinia'
import '@/styles.css'
import { AUTH_EXPIRED_EVENT } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useMenuStore } from '@/stores/menu'
import { resetMenuRoutes } from '@/router'

const app = createApp(App)
app.use(createPinia())
app.use(Alert)
app.use(Avatar)
app.use(Button)
app.use(Card)
app.use(Checkbox)
app.use(Descriptions)
app.use(Divider)
app.use(Dropdown)
app.use(Empty)
app.use(Form)
app.use(Grid)
app.use(Input)
app.use(InputNumber)
app.use(Layout)
app.use(Menu)
app.use(Modal)
app.use(Popconfirm)
app.use(Radio)
app.use(Result)
app.use(Select)
app.use(Space)
app.use(Spin)
app.use(Statistic)
app.use(Switch)
app.use(Table)
app.use(Tabs)
app.use(Tag)
app.use(Textarea)
app.use(Tooltip)
app.use(router)
app.mount('#app')

window.addEventListener(AUTH_EXPIRED_EVENT, () => {
  const auth = useAuthStore()
  auth.clear()
  useMenuStore().reset()
  resetMenuRoutes()
  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})
